package config

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"os"
	"time"
)

type Config struct {
	Debug   bool
	Spotify struct {
		ClientID     string
		ClientSecret string
		AccessToken  string
		RefreshToken string
		Expiry       time.Time
		TokenType    string
		RedirectURI  string
	}
	Tidal struct {
		UserID       string
		AccessToken  string
		RefreshToken string
	}
	Lidarr struct {
		Host   string
		APIKey string
	}
}

func Initialize() error {
	configLocation := "/data/config"
	configName := "config"
	configType := "json"
	configPath := fmt.Sprintf("%s/%s.%s", configLocation, configName, configType)

	viper.AddConfigPath("/data/config")
	viper.SetConfigName("config")
	viper.SetConfigType("json")

	viper.SetDefault("debug", false)

	viper.SetDefault("spotify.client_id", "")
	viper.SetDefault("spotify.client_secret", "")
	viper.SetDefault("spotify.access_token", "")
	viper.SetDefault("spotify.refresh_token", "")
	viper.SetDefault("spotify.expiry", "")
	viper.SetDefault("spotify.token_type", "")
	viper.SetDefault("spotify.redirect_uri", "http://localhost:28542/callback")
	viper.SetDefault("tidal.user_id", "")
	viper.SetDefault("tidal.access_token", "")
	viper.SetDefault("tidal.refresh_token", "")
	viper.SetDefault("lidarr.host", "")
	viper.SetDefault("lidarr.api_key", "")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Info().Msg("Config file not found, creating...")
		err := os.MkdirAll(configLocation, 0755)
		err = viper.SafeWriteConfigAs(configPath)
		if err != nil {
			return fmt.Errorf("error creating config file: %w", err)
		}
	} else {
		err := viper.ReadInConfig()
		if err != nil {
			return fmt.Errorf("error reading config file: %w", err)
		}
		log.Debug().Msgf("Using config file: %s", viper.ConfigFileUsed())
	}
	return nil
}
