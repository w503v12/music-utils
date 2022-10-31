package lidarr

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"io"
	"net/http"
)

func UnmarshalWanted(data []byte) (Wanted, error) {
	var r Wanted
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Wanted) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Wanted struct {
	Page          int64    `json:"page"`
	PageSize      int64    `json:"pageSize"`
	SortKey       string   `json:"sortKey"`
	SortDirection string   `json:"sortDirection"`
	TotalRecords  int64    `json:"totalRecords"`
	Records       []Record `json:"records"`
}

type Record struct {
	Title          string        `json:"title"`
	Disambiguation string        `json:"disambiguation"`
	Overview       string        `json:"overview"`
	ArtistID       int64         `json:"artistId"`
	ForeignAlbumID string        `json:"foreignAlbumId"`
	Monitored      bool          `json:"monitored"`
	AnyReleaseOk   bool          `json:"anyReleaseOk"`
	ProfileID      int64         `json:"profileId"`
	Duration       int64         `json:"duration"`
	AlbumType      string        `json:"albumType"`
	SecondaryTypes []interface{} `json:"secondaryTypes"`
	MediumCount    int64         `json:"mediumCount"`
	Ratings        Ratings       `json:"ratings"`
	ReleaseDate    string        `json:"releaseDate"`
	Releases       []Release     `json:"releases"`
	Genres         []string      `json:"genres"`
	Media          []Media       `json:"media"`
	Artist         Artist        `json:"artist"`
	Images         []Image       `json:"images"`
	Links          []Link        `json:"links"`
	Statistics     Statistics    `json:"statistics"`
	Grabbed        bool          `json:"grabbed"`
	ID             int64         `json:"id"`
}

type Artist struct {
	ArtistMetadataID  int64         `json:"artistMetadataId"`
	Status            string        `json:"status"`
	Ended             bool          `json:"ended"`
	ArtistName        string        `json:"artistName"`
	ForeignArtistID   string        `json:"foreignArtistId"`
	TadbID            int64         `json:"tadbId"`
	DiscogsID         int64         `json:"discogsId"`
	Overview          string        `json:"overview"`
	ArtistType        string        `json:"artistType"`
	Disambiguation    string        `json:"disambiguation"`
	Links             []Link        `json:"links"`
	Images            []Image       `json:"images"`
	Path              string        `json:"path"`
	QualityProfileID  int64         `json:"qualityProfileId"`
	MetadataProfileID int64         `json:"metadataProfileId"`
	Monitored         bool          `json:"monitored"`
	MonitorNewItems   string        `json:"monitorNewItems"`
	Genres            []string      `json:"genres"`
	CleanName         string        `json:"cleanName"`
	SortName          string        `json:"sortName"`
	Tags              []interface{} `json:"tags"`
	Added             string        `json:"added"`
	Ratings           Ratings       `json:"ratings"`
	Statistics        Statistics    `json:"statistics"`
	ID                int64         `json:"id"`
}

type Image struct {
	URL       string    `json:"url"`
	CoverType string    `json:"coverType"`
	Extension Extension `json:"extension"`
	RemoteURL *string   `json:"remoteUrl,omitempty"`
}

type Link struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

type Ratings struct {
	Votes float64 `json:"votes"`
	Value float64 `json:"value"`
}

type Statistics struct {
	AlbumCount      *int64  `json:"albumCount,omitempty"`
	TrackFileCount  int64   `json:"trackFileCount"`
	TrackCount      int64   `json:"trackCount"`
	TotalTrackCount int64   `json:"totalTrackCount"`
	SizeOnDisk      int64   `json:"sizeOnDisk"`
	PercentOfTracks float64 `json:"percentOfTracks"`
}

type Media struct {
	MediumNumber int64  `json:"mediumNumber"`
	MediumName   string `json:"mediumName"`
	MediumFormat Format `json:"mediumFormat"`
}

type Release struct {
	ID               int64    `json:"id"`
	AlbumID          int64    `json:"albumId"`
	ForeignReleaseID string   `json:"foreignReleaseId"`
	Title            string   `json:"title"`
	Status           Status   `json:"status"`
	Duration         int64    `json:"duration"`
	TrackCount       int64    `json:"trackCount"`
	Media            []Media  `json:"media"`
	MediumCount      int64    `json:"mediumCount"`
	Disambiguation   string   `json:"disambiguation"`
	Country          []string `json:"country"`
	Label            []string `json:"label"`
	Format           Format   `json:"format"`
	Monitored        bool     `json:"monitored"`
}

type Extension string

const (
	Jpg Extension = ".jpg"
	PNG Extension = ".png"
)

type Format string

const (
	CD           Format = "CD"
	DigitalMedia Format = "Digital Media"
	The2XCD      Format = "2xCD"
)

type Status string

const (
	Official Status = "Official"
)

type Service struct {
	Host   string
	ApiKey string
}

func InitializeService() (*Service, error) {
	host := viper.GetString("lidarr.host")
	apiKey := viper.GetString("lidarr.api_key")
	if host == "" || apiKey == "" {
		log.Error().Msg("Lidarr host or api key not set")
	}
	return &Service{
		Host:   host,
		ApiKey: apiKey,
	}, nil
}

func (s *Service) GetWanted() ([]Record, error) {
	lidarrUrl := fmt.Sprintf("%s/api/v1/wanted/missing?pageSize=10000", s.Host)

	req, err := http.NewRequest("GET", lidarrUrl, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error creating request")
		return nil, err
	}
	req.Header.Set("X-Api-Key", s.ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Error getting wanted albums")
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Error reading response body")
		return nil, err
	}
	wanted, err := UnmarshalWanted(body)
	if err != nil {
		log.Error().Err(err).Msg("Error unmarshalling response body")
		return nil, err
	}
	return wanted.Records, nil
}
