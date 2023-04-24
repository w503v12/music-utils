package spotify

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/music-utils/internal/file"
	"github.com/zmb3/spotify/v2"
)

type Service struct {
	client *spotify.Client
}

func InitializeService() (*Service, error) {
	s, err := authFlow()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) SaveUserPlaylists() error {
	// Fetch playlists
	playlists, err := s.GetUserSimplePlaylists()
	if err != nil {
		return fmt.Errorf("error getting user playlists: %w", err)
	}
	// Get more playlist information
	for _, playlist := range playlists {

		// Get full playlist
		fullPlaylist, err := s.GetPlaylist(playlist.ID)
		if err != nil {
			return fmt.Errorf("error getting playlist: %w", err)
		}

		if fullPlaylist.Name == "" {
			log.Warn().Msgf("Skipping playlist: %s as it does not have a name", playlist.ID)
			continue
		}

		// Fetch playlist tracks
		tracks, err := s.GetPlaylistTracks(playlist.ID)
		if err != nil {
			return fmt.Errorf("error getting playlist tracks: %w", err)
		}

		// FullTrack to PlaylistTrack
		var playlistTracks []spotify.PlaylistTrack
		for _, track := range tracks {
			playlistTracks = append(playlistTracks, spotify.PlaylistTrack{Track: *track})

		}
		// Set tracks
		fullPlaylist.Tracks.Tracks = playlistTracks

		// Write playlist to file
		err = file.WritePlaylistToFile(fullPlaylist)
		if err != nil {
			return fmt.Errorf("error writing playlist to file: %w", err)
		}

		log.Info().Msgf("Saved playlist: %s", fullPlaylist.Name)
	}
	return nil
}

func (s *Service) GetUserSimplePlaylists() ([]spotify.SimplePlaylist, error) {
	simplePlaylists, err := s.client.CurrentUsersPlaylists(context.Background())
	if err != nil {
		log.Error().Msgf("Error getting users playlists: %w", err)
		return nil, err
	}
	var allSimplePlaylists []spotify.SimplePlaylist
	for page := 1; ; page++ {
		{
			// Append playlists
			for _, playlist := range simplePlaylists.Playlists {
				allSimplePlaylists = append(allSimplePlaylists, playlist)
			}
			err = s.client.NextPage(context.Background(), simplePlaylists)
			if err == spotify.ErrNoMorePages {
				break
			}
			if err != nil {
				log.Error().Msgf("Error getting user playlists: %w", err)
			}
		}
	}
	return allSimplePlaylists, nil
}

func (s *Service) GetPlaylist(id spotify.ID) (*spotify.FullPlaylist, error) {
	playlist, err := s.client.GetPlaylist(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return playlist, nil
}

func (s *Service) GetPlaylistTracks(id spotify.ID) ([]*spotify.FullTrack, error) {
	items, err := s.client.GetPlaylistItems(context.Background(), id)
	if err != nil {
		return nil, err
	}
	var allPlaylistTracks []*spotify.FullTrack
	for page := 1; ; page++ {
		{
			// Append tracks
			for _, track := range items.Items {
				// Convert to FullTrack
				allPlaylistTracks = append(allPlaylistTracks, track.Track.Track)
			}
			err = s.client.NextPage(context.Background(), items)
			if err == spotify.ErrNoMorePages {
				break
			}
			if err != nil {
				log.Error().Msgf("Error getting playlist tracks: %w", err)
			}
		}
	}
	return allPlaylistTracks, nil
}
