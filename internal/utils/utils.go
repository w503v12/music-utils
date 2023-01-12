package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/music-utils/internal/tidal"
	spotifyPkg "github.com/zmb3/spotify/v2"
)

func SpotifyPlaylistOnTidal(a string, list []tidal.Playlist) (bool, int) {
	for i, b := range list {
		if strings.TrimSpace(b.Title) == strings.TrimSpace(a) {
			return true, i
		}
	}
	return false, 0
}

func spotifyTrackInTidalPlaylist(a string, list tidal.TidalPlaylistTracks) (bool, int) {
	for i, b := range list.Items {
		if strings.TrimSpace(b.Title) == strings.TrimSpace(a) {
			return true, i
		}
	}
	return false, 0
}

func SpotifyToTidalSearch(tidalService *tidal.Service, track spotifyPkg.PlaylistTrack, tidalPlaylist tidal.Playlist, tidalPlaylistTracks tidal.TidalPlaylistTracks, missingTracks *[]*spotifyPkg.PlaylistTrack) {
	// Check if track is already in Tidal playlist
	inPlaylist, _ := spotifyTrackInTidalPlaylist(track.Track.Name, tidalPlaylistTracks)
	if inPlaylist {
		log.Debug().Msgf("Track %s already in playlist %s", track.Track.Name, tidalPlaylist.Title)
		return
	}
	// Search for track on Tidal
	tidalTrack, err := tidalService.SearchTracks(fmt.Sprintf("%s %s", track.Track.Name, track.Track.Artists[0].Name))
	if err != nil {
		log.Error().Msgf("Error searching for track %s: %w", track.Track.Name, err)
		return
	}
	if len(tidalTrack.Tracks.Items) > 0 {
		var added bool
		// Check if search results contain the track ISRC code
		for _, item := range tidalTrack.Tracks.Items {
			// Case insensitive comparison
			c := strings.EqualFold(item.Isrc, track.Track.ExternalIDs["isrc"])
			if c {
				log.Debug().Msgf("Found matching track %s on Tidal", track.Track.Name)
				// Add track to Tidal playlist
				err = tidalService.AddTrackToPlaylist(tidalPlaylist.UUID, item.ID)
				if err != nil {
					log.Error().Msgf("Error adding track %s to playlist %s: %w", track.Track.Name, tidalPlaylist.Title, err)
					return
				}
				added = true
				break
			} else {
				// Begin the hell that is trying to match songs between platforms :(
				// Compare first 4 characters of ISRC
				if len(item.Isrc) < 4 || len(track.Track.ExternalIDs["isrc"]) < 4 || item.Isrc == "" || track.Track.ExternalIDs["isrc"] == "" {
					log.Info().Msgf("ISRC code for track %s is invalid", track.Track.Name)
					continue
				}
				c = strings.EqualFold(item.Isrc[:4], track.Track.ExternalIDs["isrc"][:4])
				if c {
					log.Debug().Msgf("Found matching track %s on Tidal", track.Track.Name)
					// Add track to Tidal playlist
					err = tidalService.AddTrackToPlaylist(tidalPlaylist.UUID, item.ID)
					if err != nil {
						log.Error().Err(err).Msg("Couldn't add track to Tidal playlist")
						return
					}
					added = true
					break
				}
				// Compare track name and artist with "top hit"
				title := strings.EqualFold(tidalTrack.TopHit.Value.Title, track.Track.Name)
				artist := strings.EqualFold(tidalTrack.TopHit.Value.Artists[0].Name, track.Track.Artists[0].Name)
				if title && artist {
					log.Debug().Msgf("Found matching track %s on Tidal", track.Track.Name)
					// Add track to Tidal playlist
					err = tidalService.AddTrackToPlaylist(tidalPlaylist.UUID, item.ID)
					if err != nil {
						log.Error().Err(err).Msg("Couldn't add track to Tidal playlist")
						return
					}
					added = true
					break
				}
				// Compare top hit removing and, (, or [ from title
				topHit := strings.Split(item.Title, " (")
				topHit = strings.Split(topHit[0], " [")
				artist = strings.EqualFold(item.Artists[0].Name, track.Track.Artists[0].Name)
				titleCompare := strings.EqualFold(topHit[0], track.Track.Name)
				if titleCompare && artist {
					log.Debug().Msgf("Found matching track %s on Tidal", track.Track.Name)
					// Add track to Tidal playlist
					err = tidalService.AddTrackToPlaylist(tidalPlaylist.UUID, item.ID)
					if err != nil {
						log.Error().Err(err).Msg("Couldn't add track to Tidal playlist")
						return
					}
					added = true
					break
				}
				// If last character is S, remove it and compare
				if strings.HasSuffix(track.Track.Name, "s") {
					titleCompareSuffix := strings.EqualFold(topHit[0], strings.TrimSuffix(track.Track.Name, "s"))
					if titleCompareSuffix && artist {
						log.Debug().Msgf("Found matching track %s on Tidal", track.Track.Name)
						// Add track to Tidal playlist
						err = tidalService.AddTrackToPlaylist(tidalPlaylist.UUID, item.ID)
						if err != nil {
							log.Error().Err(err).Msg("Couldn't add track to Tidal playlist")
							return
						}
						added = true
						break
					}
				}
				// If second artist is present, compare
				if len(track.Track.Artists) > 1 {
					artist = strings.EqualFold(tidalTrack.TopHit.Value.Artists[0].Name, track.Track.Artists[1].Name)
					if titleCompare && artist {
						log.Debug().Msgf("Found matching track %s on Tidal", track.Track.Name)
						// Add track to Tidal playlist
						err = tidalService.AddTrackToPlaylist(tidalPlaylist.UUID, item.ID)
						if err != nil {
							log.Error().Err(err).Msg("Couldn't add track to Tidal playlist")
							return
						}
						added = true
						break
					}
				}
			}

		}
		if !added {
			// Add track to missing tracks
			*missingTracks = append(*missingTracks, &track)
		}
	} else {
		// Add track to missing tracks
		*missingTracks = append(*missingTracks, &track)
	}
}

func ExtractUUID(url string) string {
	// Use regex to extract UUID from URL
	re := regexp.MustCompile(`(?m)(?i)([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})`)
	return re.FindString(url)
}

func JoinWithCommasAnd(items []string) string {
	if len(items) <= 1 {
		return strings.Join(items, "")
	}
	last := items[len(items)-1]
	items = items[:len(items)-1]
	return fmt.Sprintf("%s and %s", strings.Join(items, ", "), last)
}
