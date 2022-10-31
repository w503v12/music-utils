package main

import (
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/spf13/viper"
	"github.com/zibbp/music-utils/internal/config"
	"github.com/zibbp/music-utils/internal/file"
	"github.com/zibbp/music-utils/internal/lidarr"
	"github.com/zibbp/music-utils/internal/navidrome"
	"github.com/zibbp/music-utils/internal/spotify"
	"github.com/zibbp/music-utils/internal/tidal"
	"github.com/zibbp/music-utils/internal/utils"
	spotifyPkg "github.com/zmb3/spotify/v2"
)

type Playlists struct {
	Playlists []Playlist `json:"playlists"`
}

type Playlist struct {
	Spotify spotifyPkg.FullPlaylist `json:"spotify"`
	Tidal   tidal.Playlist          `json:"tidal"`
}

func main() {

	// Config
	err := config.Initialize()
	if err != nil {
		log.Error().Msgf("Error initializing config: %w", err)
	}

	// Files
	err = file.Initialize()
	if err != nil {
		log.Error().Msgf("Error initializing file service: %w", err)
	}

	// Logging
	configDebug := viper.GetBool("debug")
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	if configDebug {
		log.Info().Msg("debug mode enabled")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Flags
	saveSpotifyFlag := flag.Bool("save-spotify", false, "Save Spotify playlists to files")
	toTidalFlag := flag.Bool("to-tidal", false, "Imports Spotify playlists to Tidal")
	importNavidromeFlag := flag.Bool("import-navidrome", false, "Generates Navidrome playlist files from Tidal using Navidrome's database")
	saveTidalFlag := flag.Bool("save-tidal", false, "Save provided Tidal playlists to files")
	processLidarrWanted := flag.Bool("process-lidarr-wanted", false, "Process Lidarr wanted albums")
	flag.Parse()

	if *saveSpotifyFlag {
		log.Info().Msg("save-spotify flag enabled")
		// Create Spotify service
		spotifyService, err := spotify.InitializeService()
		if err != nil {
			log.Fatal().Msgf("Error initializing spotify service: %w", err)
		}
		err = spotifyService.SaveUserPlaylists()
		if err != nil {
			log.Error().Msgf("Error saving user playlists: %w", err)
		}
	}

	if *toTidalFlag {
		// Tidal service
		tidalService, err := tidal.InitializeService()
		if err != nil {
			log.Fatal().Msgf("Error initializing tidal service: %w", err)
		}
		// Read local Spotify playlists from files
		spotifyPlaylists, err := file.ReadUsersPlaylists()
		if err != nil {
			log.Fatal().Msgf("Error reading users playlists: %w", err)
		}
		if len(spotifyPlaylists) == 0 {
			log.Fatal().Msg("No Spotify playlists found")
		}
		log.Info().Msgf("Found %d Spotify playlists to process", len(spotifyPlaylists))
		tidalPlaylists, err := tidalService.GetUserPlaylists()
		if err != nil {
			log.Fatal().Msgf("Error getting user tidal playlists: %w", err)
		}
		log.Info().Msgf("Found %d Tidal playlists", len(tidalPlaylists.Items))

		var playlists Playlists
		// Check if Spotify playlists exists on Tidal
		for _, spotifyPlaylist := range spotifyPlaylists {
			var playlist Playlist
			onTidal, i := utils.SpotifyPlaylistOnTidal(spotifyPlaylist.Name, tidalPlaylists.Items)
			if !onTidal {
				log.Info().Msgf("Playlist %s not found on Tidal", spotifyPlaylist.Name)
				// Create playlist on Tidal
				tidalPlaylist, err := tidalService.CreatePlaylist(spotifyPlaylist.Name, spotifyPlaylist.Description)
				if err != nil {
					log.Error().Msgf("Error creating playlist %s on Tidal: %w", spotifyPlaylist.Name, err)
				}
				log.Info().Msgf("Created playlist %s on Tidal", tidalPlaylist.Title)

				// Fetch full playlist
				fullTidalPlaylist, err := tidalService.GetPlaylist(tidalPlaylist.UUID)
				if err != nil {
					log.Error().Msgf("Error fetching playlist %s from Tidal: %w", tidalPlaylist.Title, err)
				}
				playlist.Tidal = fullTidalPlaylist
				playlist.Spotify = spotifyPlaylist
				playlists.Playlists = append(playlists.Playlists, playlist)
			} else {
				log.Debug().Msgf("Playlist %s found on Tidal", spotifyPlaylist.Name)
				// Fetch full Tidal playlist
				fullTidalPlaylist, err := tidalService.GetPlaylist(tidalPlaylists.Items[i].UUID)
				if err != nil {
					log.Error().Msgf("Error fetching playlist %s from Tidal: %w", tidalPlaylists.Items[i].Title, err)
				}
				playlist.Tidal = fullTidalPlaylist
				playlist.Spotify = spotifyPlaylist
				playlists.Playlists = append(playlists.Playlists, playlist)
			}
		}

		// Spotify to Tidal import
		for _, playlist := range playlists.Playlists {
			log.Info().Msgf("Importing playlist %s to Tidal", playlist.Spotify.Name)
			// Get all Tidal tracks
			tidalPlaylistTracks, err := tidalService.GetPlaylistTracks(playlist.Tidal.UUID)
			if err != nil {
				log.Error().Msgf("Error getting playlist tracks for %s: %w", playlist.Tidal.Title, err)
			}
			// Check if tracks exist on Tidal
			var missingTracks []*spotifyPkg.PlaylistTrack
			for _, spotifyTrack := range playlist.Spotify.Tracks.Tracks {
				// Spotify edge case if track is missing
				if spotifyTrack.Track.ID == "" {
					log.Debug().Msgf("Track %s is missing ID", spotifyTrack.Track.Name)
					continue
				}
				utils.SpotifyToTidalSearch(tidalService, spotifyTrack, playlist.Tidal, tidalPlaylistTracks, &missingTracks)
			}
			// Missing tracks
			if len(missingTracks) > 0 {
				log.Info().Msgf("Found %d missing tracks", len(missingTracks))
				err := file.ProcessMissingTracks(missingTracks, playlist.Spotify.Name)
				if err != nil {
					log.Error().Msgf("Error processing missing tracks: %w", err)
					return
				}
			}
			// Fetch Tidal playlist and write to file
			tidalPlaylistTracks, err = tidalService.GetPlaylistTracks(playlist.Tidal.UUID)
			if err != nil {
				log.Error().Msgf("Error getting playlist tracks for %s: %w", playlist.Tidal.Title, err)
				return
			}
			playlist.Tidal.Tracks = tidalPlaylistTracks.Items
			// Write to file
			err = file.WriteTidalPlaylistToFile(playlist.Tidal)
			if err != nil {
				log.Error().Msgf("Error writing tidal playlist to file: %w", err)
				return
			}
			log.Info().Msgf("Finished importing playlist %s to Tidal", playlist.Spotify.Name)
		}
	}

	if *saveTidalFlag {
		// Tidal service
		tidalService, err := tidal.InitializeService()
		if err != nil {
			log.Fatal().Msgf("Error initializing tidal service: %w", err)
		}
		log.Info().Msg("Saving Tidal playlists to file")
		playlistUrls, err := file.ReadTidalPlaylistsToSave()
		if err != nil {
			log.Fatal().Msgf("Error reading tidal playlists to save: %w", err)
		}
		for _, playlistUrl := range playlistUrls {
			// Extract uuid from url
			uuid := utils.ExtractUUID(playlistUrl)
			if uuid == "" {
				log.Error().Msgf("Error extracting uuid from %s", playlistUrl)
				continue
			}
			// Get playlist
			tidalPlaylist, err := tidalService.GetPlaylist(uuid)
			if err != nil {
				log.Error().Msgf("Error getting playlist %s from Tidal: %w", uuid, err)
				continue
			}
			// Get playlist tracks
			tidalPlaylistTracks, err := tidalService.GetPlaylistTracks(uuid)
			if err != nil {
				log.Error().Msgf("Error getting playlist tracks for %s: %w", tidalPlaylist.Title, err)
				continue
			}
			tidalPlaylist.Tracks = tidalPlaylistTracks.Items

			// Write to file
			err = file.WriteTidalPlaylistToFile(tidalPlaylist)
			if err != nil {
				log.Error().Msgf("Error writing tidal playlist to file: %w", err)
				return
			}
			log.Info().Msgf("Finished saving playlist %s to file", tidalPlaylist.Title)
		}
	}

	if *importNavidromeFlag {
		navidromeService, err := navidrome.InitializeService()
		if err != nil {
			log.Fatal().Msgf("Error initializing navidrome service: %w", err)
		}
		log.Info().Msg("Starting Navidrome import")
		// Read Tidal playlist files
		tidalPlaylists, err := file.ReadTidalPlaylists()
		if err != nil {
			log.Fatal().Msgf("Error reading tidal playlists: %w", err)
		}
		log.Info().Msgf("Found %d Tidal playlists to import", len(tidalPlaylists))

		for _, tidalPlaylist := range tidalPlaylists {
			// Create m3u8 file
			err := file.CreateM3U8PlaylistFile(tidalPlaylist.Title)
			if err != nil {
				log.Error().Msgf("Error creating m3u8 file: %w", err)
			}
			log.Info().Msgf("Processing playlist %s which has %d tracks", tidalPlaylist.Title, len(tidalPlaylist.Tracks))
			// Loop tracks
			var missingTracks []tidal.Track
			for _, track := range tidalPlaylist.Tracks {
				foundTrack, err := navidromeService.Db.FindTrack(track.Title, track.Album.Title, track.Artist.Name)
				if err != nil {
					log.Debug().Msgf("Error finding track %s: %w", track.Title, err)
				}
				if foundTrack != "" {
					log.Debug().Msgf("Found track %s", track.Title)
					// Add track to m3u8 file
					err := file.AddTrackToM3U8PlaylistFile(tidalPlaylist.Title, foundTrack)
					if err != nil {
						log.Error().Msgf("Error adding track to m3u8 file: %w", err)
					}
				} else {
					log.Debug().Msgf("Track %s not found", track.Title)
					missingTracks = append(missingTracks, track)
				}
			}
			// Missing tracks
			if len(missingTracks) > 0 {
				log.Info().Msgf("Found %d missing tracks", len(missingTracks))
				err := file.ProcessMissingNavidromeTracks(missingTracks, tidalPlaylist.Title)
				if err != nil {
					log.Error().Msgf("Error processing missing tracks: %w", err)
					return
				}
			}
			log.Info().Msgf("Finished processing playlist %s", tidalPlaylist.Title)
		}

	}

	if *processLidarrWanted {
		lidarrService, err := lidarr.InitializeService()
		if err != nil {
			log.Fatal().Msgf("Error initializing lidarr service: %w", err)
		}
		tidalService, err := tidal.InitializeService()
		if err != nil {
			log.Fatal().Msgf("Error initializing tidal service: %w", err)
		}
		wantedAlbums, err := lidarrService.GetWanted()
		if err != nil {
			log.Fatal().Msgf("Error getting wanted records: %w", err)
		}
		log.Info().Msgf("Found %d wanted albums", len(wantedAlbums))

		var links []string
		var missingAlbums []lidarr.Record
		// Process each wanted album
		for _, wantedAlbum := range wantedAlbums {
			// Find album on Tidal
			tidalAlbum, err := tidalService.FindAlbum(wantedAlbum.Title, wantedAlbum.Artist.ArtistName)
			if err != nil {
				log.Error().Msgf("Error finding album %s: %w", wantedAlbum.Title, err)
				missingAlbums = append(missingAlbums, wantedAlbum)
				continue
			}
			if len(tidalAlbum.Albums.Items) == 0 {
				log.Error().Msgf("Could not find album %s", wantedAlbum.Title)
				missingAlbums = append(missingAlbums, wantedAlbum)
				continue
			}
			// Compare number of tracks
			if tidalAlbum.Albums.Items[0].NumberOfTracks != wantedAlbum.Statistics.TrackCount {
				log.Error().Msgf("Number of tracks for album %s does not match", wantedAlbum.Title)
				missingAlbums = append(missingAlbums, wantedAlbum)
				continue
			}
			links = append(links, tidalAlbum.Albums.Items[0].URL)
		}
		// Write links to file
		err = file.WriteWantedLinks(links)
		if err != nil {
			log.Error().Msgf("Error writing wanted links to file: %w", err)
			return
		}
		// Process missing albums
		if len(missingAlbums) > 0 {
			log.Info().Msgf("Found %d missing albums", len(missingAlbums))
			err := file.ProcessMissingLidarrAlbums(missingAlbums)
			if err != nil {
				log.Error().Msgf("Error processing missing albums: %w", err)
				return
			}
		}
		log.Info().Msgf("Finished writing wanted links to file")
	}
}
