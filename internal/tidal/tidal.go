package tidal

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	apiURL      = "https://listen.tidal.com/v1"
	apiURL2     = "https://listen.tidal.com/v2"
	countryCode = "US"
)

type Service struct {
	UserID       string
	AccessToken  string
	RefreshToken string
	ClientID     string
	ClientSecret string
}

type CreatedPlaylist struct {
	Trn            string      `json:"trn"`
	ItemType       string      `json:"itemType"`
	AddedAt        string      `json:"addedAt"`
	LastModifiedAt string      `json:"lastModifiedAt"`
	Name           string      `json:"name"`
	Parent         interface{} `json:"parent"`
	Data           Playlist    `json:"data"`
}

type Playlist struct {
	UUID            string           `json:"uuid"`
	Title           string           `json:"title"`
	NumberOfTracks  int64            `json:"numberOfTracks"`
	NumberOfVideos  int64            `json:"numberOfVideos"`
	Creator         Creator          `json:"creator"`
	Description     string           `json:"description"`
	Duration        int64            `json:"duration"`
	LastUpdated     string           `json:"lastUpdated"`
	Created         string           `json:"created"`
	Type            string           `json:"type"`
	PublicPlaylist  bool             `json:"publicPlaylist"`
	URL             string           `json:"url"`
	Image           string           `json:"image"`
	Popularity      int64            `json:"popularity"`
	SquareImage     string           `json:"squareImage"`
	PromotedArtists []PromotedArtist `json:"promotedArtists"`
	LastItemAddedAt string           `json:"lastItemAddedAt"`
	Tracks          []Track          `json:"tracks"`
}

type Creator struct {
	ID int64 `json:"id"`
}

type PromotedArtist struct {
	ID      int64       `json:"id"`
	Name    string      `json:"name"`
	Type    string      `json:"type"`
	Picture interface{} `json:"picture"`
}

type TidalPlaylistTracks struct {
	Limit              int64   `json:"limit"`
	Offset             int64   `json:"offset"`
	TotalNumberOfItems int64   `json:"totalNumberOfItems"`
	Items              []Track `json:"items"`
}

type Track struct {
	ID                   int64       `json:"id"`
	Title                string      `json:"title"`
	Duration             int64       `json:"duration"`
	ReplayGain           float64     `json:"replayGain"`
	Peak                 float64     `json:"peak"`
	AllowStreaming       bool        `json:"allowStreaming"`
	StreamReady          bool        `json:"streamReady"`
	StreamStartDate      *string     `json:"streamStartDate"`
	PremiumStreamingOnly bool        `json:"premiumStreamingOnly"`
	TrackNumber          int64       `json:"trackNumber"`
	VolumeNumber         int64       `json:"volumeNumber"`
	Version              *string     `json:"version"`
	Popularity           int64       `json:"popularity"`
	Copyright            string      `json:"copyright"`
	Description          interface{} `json:"description"`
	URL                  string      `json:"url"`
	Isrc                 string      `json:"isrc"`
	Editable             bool        `json:"editable"`
	Explicit             bool        `json:"explicit"`
	AudioQuality         string      `json:"audioQuality"`
	AudioModes           []string    `json:"audioModes"`
	Artist               Artist      `json:"artist"`
	Artists              []Artist    `json:"artists"`
	Album                Album       `json:"album"`
	Mixes                Mixes       `json:"mixes"`
	DateAdded            string      `json:"dateAdded"`
	Index                int64       `json:"index"`
	ItemUUID             string      `json:"itemUuid"`
}

type Album struct {
	ID           int64   `json:"id"`
	Title        string  `json:"title"`
	Cover        string  `json:"cover"`
	VibrantColor string  `json:"vibrantColor"`
	VideoCover   *string `json:"videoCover"`
	ReleaseDate  string  `json:"releaseDate"`
}

type Artist struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Picture string `json:"picture"`
}

type Mixes struct {
	MasterTrackMix *string `json:"MASTER_TRACK_MIX,omitempty"`
	TrackMix       string  `json:"TRACK_MIX"`
}

type TrackSearch struct {
	Artists   SearchTracksPagination `json:"artists"`
	Albums    SearchTracksPagination `json:"albums"`
	Playlists SearchTracksPagination `json:"playlists"`
	Tracks    SearchTracksPagination `json:"tracks"`
	Videos    SearchTracksPagination `json:"videos"`
	TopHit    TopHit                 `json:"topHit"`
}

type SearchTracksPagination struct {
	Limit              int64   `json:"limit"`
	Offset             int64   `json:"offset"`
	TotalNumberOfItems int64   `json:"totalNumberOfItems"`
	Items              []Track `json:"items"`
}

type UserPlaylists struct {
	Limit              int64      `json:"limit"`
	Offset             int64      `json:"offset"`
	TotalNumberOfItems int64      `json:"totalNumberOfItems"`
	Items              []Playlist `json:"items"`
}

type TopHit struct {
	Value Track  `json:"value"`
	Type  string `json:"type"`
}

func InitializeService() (*Service, error) {
	log.Info().Msg("Initializing Tidal service...")
	var s Service
	s.ClientID = clientId
	s.ClientSecret = clientSecret

	if viper.GetString("tidal.access_token") == "" || viper.GetString("tidal.refresh_token") == "" {
		log.Debug().Msg("No Tidal access token or refresh token found in config, attempting to get new tokens...")
		deviceCode, err := getDeviceCode()
		if err != nil {
			log.Fatal().Err(err).Msg("Error getting device code")
			return nil, err
		}
		log.Info().Msgf("Please visit the following URL to authorize this application: https://%v", deviceCode.VerificationURIComplete)

		// Begin polling for authorization
		for {
			loginResponse, err := tokenLogin(deviceCode)
			if err != nil {
				log.Fatal().Err(err).Msg("Tidal auth failed at token login")
			}
			if (AuthLogin{} == loginResponse.AuthLogin) {
				// No auth token - check what errors occurred
				// If error is expired_token, the device ID expired (5 minutes)
				if loginResponse.AuthError.Error == "expired_token" {
					log.Fatal().Msg("Tidal auth failed at token login - device ID expired. Please try again")
				}
			} else {
				// Auth token received - break loop
				log.Info().Msg("Tidal auth successful")
				s.UserID = strconv.Itoa(int(loginResponse.AuthLogin.User.UserID))
				s.AccessToken = loginResponse.AuthLogin.AccessToken
				s.RefreshToken = loginResponse.AuthLogin.RefreshToken
				viper.Set("tidal.access_token", s.AccessToken)
				viper.Set("tidal.refresh_token", s.RefreshToken)
				viper.Set("tidal.user_id", s.UserID)
				err := viper.WriteConfig()
				if err != nil {
					log.Fatal().Err(err).Msg("Error writing config file")
					return nil, err
				}
				break
			}
			d := time.Duration(deviceCode.Interval) * time.Second
			log.Debug().Msgf("Waiting %d seconds before trying again.", deviceCode.Interval)
			time.Sleep(d)
		}
	} else {
		configAccessToken := viper.GetString("tidal.access_token")
		configRefreshToken := viper.GetString("tidal.refresh_token")
		// Check if access token is valid
		session, err := checkSession(configAccessToken)
		if err != nil {
			log.Info().Msg("failed to get Tidal session, attempting to refresh token")
			// Get session failed. Access token is probably expired.
			refresh, err := refreshAccessToken(configRefreshToken)
			if err != nil {
				log.Error().Msg("Tidal auth failed at refreshing access token. Please log in again.")
				viper.Set("tidal.access_token", "")
				viper.Set("tidal.refresh_token", "")
				err := viper.WriteConfig()
				if err != nil {
					log.Fatal().Err(err).Msg("Error writing config file")
					return nil, err
				}
				InitializeService()
			}
			// Write new access token to config
			viper.Set("tidal.access_token", refresh.AccessToken)
			err = viper.WriteConfig()
			if err != nil {
				return nil, err
			}
			s.AccessToken = refresh.AccessToken
			s.RefreshToken = configRefreshToken
			s.UserID = strconv.Itoa(int(refresh.User.UserID))
			return &s, nil
		}
		log.Info().Msg("Tidal access token is valid")
		s.AccessToken = configAccessToken
		s.RefreshToken = configRefreshToken
		s.UserID = strconv.Itoa(int(session.UserID))
		return &s, nil
	}
	return &s, nil

}

func (s *Service) standardHttpGetRequest(reqUrl string) ([]byte, error) {
	log.Debug().Msgf("Tidal GET request: %v", reqUrl)

	client := &http.Client{}

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, err
	}

	// Set Headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AccessToken))

	// Set Query Params
	q := url.Values{}
	q.Add("countryCode", countryCode)
	q.Add("limit", "10000")

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", string(body))
	}

	return body, nil
}

func (s *Service) GetUserPlaylists() (UserPlaylists, error) {
	log.Debug().Msgf("Getting playlists for Tidal user %s", s.UserID)
	playlists, err := s.standardHttpGetRequest(fmt.Sprintf("%s/users/%s/playlists", apiURL, s.UserID))
	if err != nil {
		return UserPlaylists{}, err
	}

	var tidalUserPlaylists UserPlaylists
	err = json.Unmarshal(playlists, &tidalUserPlaylists)
	if err != nil {
		return UserPlaylists{}, err
	}

	return tidalUserPlaylists, nil

}

func (s *Service) CreatePlaylist(name string, description string) (Playlist, error) {
	log.Debug().Msgf("Creating playlist %s", name)

	// HTTP
	client := &http.Client{}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/my-collection/playlists/folders/create-playlist", apiURL2), nil)
	if err != nil {
		return Playlist{}, err
	}

	// Set Headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AccessToken))

	// Set Query Params
	q := url.Values{}
	q.Add("folderId", "root")
	q.Add("name", name)
	q.Add("description", description)

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return Playlist{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Playlist{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Playlist{}, fmt.Errorf("%s", string(body))
	}

	var createdPlaylist CreatedPlaylist

	err = json.Unmarshal(body, &createdPlaylist)
	if err != nil {
		return Playlist{}, err
	}

	return createdPlaylist.Data, nil
}

func (s *Service) GetPlaylist(id string) (Playlist, error) {

	body, err := s.standardHttpGetRequest(fmt.Sprintf("%s/playlists/%s", apiURL, id))
	if err != nil {
		return Playlist{}, err
	}

	var playlist Playlist
	err = json.Unmarshal(body, &playlist)
	if err != nil {
		return Playlist{}, err
	}

	return playlist, nil
}

func (s *Service) GetPlaylistTracks(id string) (TidalPlaylistTracks, error) {
	log.Debug().Msgf("Getting playlist tracks for %s", id)

	body, err := s.standardHttpGetRequest(fmt.Sprintf("%s/playlists/%s/tracks", apiURL, id))
	if err != nil {
		return TidalPlaylistTracks{}, err
	}

	var playlistTracks TidalPlaylistTracks
	err = json.Unmarshal(body, &playlistTracks)
	if err != nil {
		return TidalPlaylistTracks{}, err
	}

	return playlistTracks, nil

}

func (s *Service) SearchTracks(query string) (TrackSearch, error) {
	log.Debug().Msgf("Searching Tidal tracks for %s", query)

	// HTTP
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/search", apiURL), nil)
	if err != nil {
		return TrackSearch{}, err
	}

	// Set Headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AccessToken))

	// Set Query Params
	q := url.Values{}
	q.Add("countryCode", countryCode)
	q.Add("limit", "20")
	q.Add("query", query)
	q.Add("types", "TRACKS")

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return TrackSearch{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TrackSearch{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return TrackSearch{}, fmt.Errorf("%s", string(body))
	}

	var trackSearch TrackSearch

	err = json.Unmarshal(body, &trackSearch)
	if err != nil {
		return TrackSearch{}, err
	}

	return trackSearch, nil

}

func (s *Service) getPlaylistEtag(id string) (string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/playlists/%s", apiURL, id), nil)
	if err != nil {
		return "", err
	}

	// Set Headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AccessToken))

	// Set Query Params
	q := url.Values{}
	q.Add("countryCode", countryCode)

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", string(body))
	}

	playlistEtag := resp.Header.Get("ETag")

	return playlistEtag, nil
}

func (s *Service) AddTrackToPlaylist(playlistId string, trackId int64) error {
	log.Debug().Msgf("Adding track %v to playlist %s", trackId, playlistId)
	playlistEtag, err := s.getPlaylistEtag(playlistId)
	if err != nil {
		return err
	}

	client := &http.Client{}

	data := url.Values{}
	data.Set("trackIds", fmt.Sprintf("%v", trackId))
	data.Set("onArtifactNotFound", "FAIL")
	data.Set("onDupes", "FAIL")

	// reqBody := fmt.Sprintf(`{"trackIds":"%v","onArtifactNotFound":"FAIL","onDupes":"FAIL"}`, trackId)
	encodedData := data.Encode()

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/playlists/%s/items", apiURL, playlistId), strings.NewReader(encodedData))
	if err != nil {
		return err
	}

	// Set Headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AccessToken))
	req.Header.Set("If-None-Match", playlistEtag)

	// Set Query Params
	q := url.Values{}
	q.Add("countryCode", countryCode)

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusConflict {
			log.Debug().Msgf("Track %v already exists in playlist %s", trackId, playlistId)
		} else {
			return fmt.Errorf("%s", string(body))
		}

		return err
	}

	return nil
}
