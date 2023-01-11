package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
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
	Notification struct {
		Webhook struct {
			Enabled bool
			URL     string
		}
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
	viper.SetDefault("notification.webhook.enabled", false)
	viper.SetDefault("notification.webhook.url", "")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Info().Msg("Config file not found, creating...")
		err := os.MkdirAll(configLocation, 0755)
		err = viper.SafeWriteConfigAs(configPath)
		if err != nil {
			return fmt.Errorf("error creating config file: %w", err)
		}
	} else {
		err := viper.ReadInConfig()
		refreshConfig(configPath)
		if err != nil {
			return fmt.Errorf("error reading config file: %w", err)
		}
		log.Debug().Msgf("Using config file: %s", viper.ConfigFileUsed())
	}
	return nil
}

func refreshConfig(configPath string) {
	if !viper.IsSet("notification.webhook.url") {
		viper.Set("notification.webhook.url", "")
	}
	if !viper.IsSet("notification.webhook.enabled") {
		viper.Set("notification.webhook.enabled", false)
	}
}

func unset(vars ...string) error {
	cfg := viper.AllSettings()
	vals := cfg

	for _, v := range vars {
		parts := strings.Split(v, ".")
		for i, k := range parts {
			v, ok := vals[k]
			if !ok {
				// Doesn't exist no action needed
				break
			}

			switch len(parts) {
			case i + 1:
				// Last part so delete.
				delete(vals, k)
			default:
				m, ok := v.(map[string]interface{})
				if !ok {
					return fmt.Errorf("unsupported type: %T for %q", v, strings.Join(parts[0:i], "."))
				}
				vals = m
			}
		}
	}

	b, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		return err
	}

	if err = viper.ReadConfig(bytes.NewReader(b)); err != nil {
		return err
	}

	return viper.WriteConfig()
}
