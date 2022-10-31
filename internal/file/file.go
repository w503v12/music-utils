package file

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kennygrant/sanitize"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/music-utils/internal/lidarr"
	"github.com/zibbp/music-utils/internal/tidal"
	"github.com/zmb3/spotify/v2"
	spotifyPkg "github.com/zmb3/spotify/v2"
	"os"
	"strings"
)

type MissingTrack struct {
	Name    string                    `json:"name"`
	Album   string                    `json:"album"`
	Artists []spotifyPkg.SimpleArtist `json:"artists"`
}

type MissingTrackNavidrome struct {
	Name    string         `json:"name"`
	Album   string         `json:"album"`
	Artists []tidal.Artist `json:"artists"`
}

type MissingLidarrAlbum struct {
	Name   string `json:"name"`
	Artist string `json:"artist"`
}

func Initialize() error {

	err := createFolderIfNotExists("/data/spotify")
	if err != nil {
		return err
	}
	err = createFolderIfNotExists("/data/missing")
	if err != nil {
		return err
	}
	err = createFolderIfNotExists("/data/tidal")
	if err != nil {
		return err
	}
	err = createFolderIfNotExists("/data/navidrome-missing")
	if err != nil {
		return err
	}
	err = createFolderIfNotExists("/data/wanted")
	if err != nil {
		return err
	}

	return nil
}

func createFolderIfNotExists(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func WriteFile(path string, data []byte) error {
	err := os.WriteFile(path, []byte(data), 0644)
	if err != nil {
		return err
	}
	return nil
}

func WritePlaylistToFile(playlist *spotify.FullPlaylist) error {
	data, err := JSONMarshal(playlist)
	if err != nil {
		return fmt.Errorf("error marshalling playlist: %w", err)
	}

	// Sanitize playlist name
	playlistName := sanitize.BaseName(playlist.Name)

	err = WriteFile(fmt.Sprintf("/data/spotify/%s.json", playlistName), data)
	if err != nil {
		return fmt.Errorf("error writing playlist file: %w", err)
	}

	return nil
}

func WriteTidalPlaylistToFile(playlist tidal.Playlist) error {
	data, err := JSONMarshal(playlist)
	if err != nil {
		return fmt.Errorf("error marshalling playlist: %w", err)
	}

	// Sanitize playlist name
	playlistName := sanitize.BaseName(playlist.Title)

	err = WriteFile(fmt.Sprintf("/data/tidal/%s.json", playlistName), data)
	if err != nil {
		return fmt.Errorf("error writing playlist file: %w", err)
	}

	return nil
}

func ProcessMissingTracks(missingTracks []*spotifyPkg.PlaylistTrack, playlistName string) error {
	// Convert to simpler track struct
	var tracks []MissingTrack
	for _, track := range missingTracks {
		newTrack := MissingTrack{
			Name:  track.Track.Name,
			Album: track.Track.Album.Name,
		}
		for _, artist := range track.Track.Artists {
			newTrack.Artists = append(newTrack.Artists, artist)
		}
		tracks = append(tracks, newTrack)
	}
	err := WriteMissingTracks(tracks, playlistName)
	if err != nil {
		return err
	}
	return nil
}

func WriteMissingTracks(tracks []MissingTrack, name string) error {
	data, err := JSONMarshal(tracks)
	if err != nil {
		fmt.Println(err)
	}

	// Sanitize playlist name
	playlistName := sanitize.BaseName(name)

	err = WriteFile(fmt.Sprintf("/data/missing/%s.json", playlistName), data)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

func ReadUsersPlaylists() ([]spotify.FullPlaylist, error) {
	// Read all playlist files
	files, err := os.ReadDir("/data/spotify")
	if err != nil {
		return nil, fmt.Errorf("error reading playlist files: %w", err)
	}

	// Create slice to hold all playlists
	var playlists []spotify.FullPlaylist

	// Loop through all files
	for _, file := range files {
		// Skip playlists.json
		if file.Name() == "playlists.json" {
			continue
		}

		// Read file
		data, err := os.ReadFile(fmt.Sprintf("/data/spotify/%s", file.Name()))
		if err != nil {
			return nil, fmt.Errorf("error reading playlist file: %w", err)
		}

		// Unmarshal file
		var playlist spotify.FullPlaylist
		err = json.Unmarshal(data, &playlist)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling playlist file: %w", err)
		}

		// Append playlist to slice
		playlists = append(playlists, playlist)
	}

	return playlists, nil
}

func ReadTidalPlaylists() ([]tidal.Playlist, error) {
	// Read all playlist files
	files, err := os.ReadDir("/data/tidal")
	if err != nil {
		return nil, fmt.Errorf("error reading playlist files: %w", err)
	}

	// Create slice to hold all playlists
	var playlists []tidal.Playlist

	// Loop through all files
	for _, file := range files {
		if file.Name() == "playlists.txt" {
			continue
		}

		// Skip playlists.json
		if file.Name() == "playlists.json" {
			continue
		}

		// Read file
		data, err := os.ReadFile(fmt.Sprintf("/data/tidal/%s", file.Name()))
		if err != nil {
			return nil, fmt.Errorf("error reading playlist file: %w", err)
		}

		// Unmarshal file
		var playlist tidal.Playlist
		err = json.Unmarshal(data, &playlist)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling playlist file: %w", err)
		}

		// Append playlist to slice
		playlists = append(playlists, playlist)
	}

	return playlists, nil
}

func CreateM3U8PlaylistFile(name string) error {
	// Check if file exists, if not create it
	playlistName := sanitize.BaseName(name)
	filePath := fmt.Sprintf("/playlists/%s.m3u8", playlistName)
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("error creating playlist file: %w", err)
		}
		if _, err := file.WriteString("#EXTM3U\n"); err != nil {
			return fmt.Errorf("error writing playlist file: %w", err)
		}
		defer file.Close()
	}
	return nil
}

func AddTrackToM3U8PlaylistFile(name string, trackPath string) error {
	// Append track to playlist file is not already in it
	playlistName := sanitize.BaseName(name)
	filePath := fmt.Sprintf("/playlists/%s.m3u8", playlistName)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("error opening playlist file: %w", err)
	}
	defer file.Close()

	// Check if track is already in playlist
	// Read file to string
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading playlist file: %w", err)
	}
	// Data to string
	dataString := string(data)
	if !strings.Contains(dataString, trackPath) {
		log.Debug().Msgf("Adding track %s to playlist %s", trackPath, playlistName)
		if _, err = file.WriteString(fmt.Sprintf("%s\n", trackPath)); err != nil {
			return fmt.Errorf("error writing playlist file: %w", err)
		}
	}
	return nil
}

func ReadTidalPlaylistsToSave() ([]string, error) {
	// Read playlists.txt
	data, err := os.ReadFile("/data/tidal/playlists.txt")
	if err != nil {
		return nil, fmt.Errorf("error reading playlists.txt: %w", err)
	}
	// Split into slice
	playlists := strings.Split(string(data), "\n")
	// Remove last item if empty
	if playlists[len(playlists)-1] == "" {
		playlists = playlists[:len(playlists)-1]
	}
	return playlists, nil
}

func ProcessMissingNavidromeTracks(missingTracks []tidal.Track, playlistName string) error {
	// Convert to simpler track struct
	var tracks []MissingTrackNavidrome
	for _, track := range missingTracks {
		newTrack := MissingTrackNavidrome{
			Name:  track.Title,
			Album: track.Album.Title,
		}
		for _, artist := range track.Artists {
			newTrack.Artists = append(newTrack.Artists, artist)
		}
		tracks = append(tracks, newTrack)
	}
	err := WriteMissingNavidromeTracks(tracks, playlistName)
	if err != nil {
		return err
	}
	return nil
}

func WriteMissingNavidromeTracks(tracks []MissingTrackNavidrome, name string) error {
	data, err := JSONMarshal(tracks)
	if err != nil {
		fmt.Println(err)
	}

	// Sanitize playlist name
	playlistName := sanitize.BaseName(name)

	err = WriteFile(fmt.Sprintf("/data/navidrome-missing/%s.json", playlistName), data)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

func WriteWantedLinks(links []string) error {
	// Write array of strings to text file
	data := strings.Join(links, "\n")
	err := WriteFile("/data/wanted/tidal.txt", []byte(data))
	if err != nil {
		return fmt.Errorf("error writing wanted links file: %w", err)
	}
	return nil
}

func ProcessMissingLidarrAlbums(albums []lidarr.Record) error {
	// Convert to a simpler struct
	var missingLidarAlbums []MissingLidarrAlbum
	for _, album := range albums {
		newAlbum := MissingLidarrAlbum{
			Name:   album.Title,
			Artist: album.Artist.ArtistName,
		}
		missingLidarAlbums = append(missingLidarAlbums, newAlbum)
	}
	data, err := JSONMarshal(missingLidarAlbums)
	if err != nil {
		return fmt.Errorf("error marshalling missing lidarr albums: %w", err)
	}
	err = WriteFile("/data/wanted/missing-albums.json", data)
	if err != nil {
		return fmt.Errorf("error writing missing lidarr albums file: %w", err)
	}
	return nil
}

// JSONMarshal is a wrapper for json.Marshal which does not escape unicode characters (&)
func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}
