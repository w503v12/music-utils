package spotify

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
	"net/http"
)

var (
	ch    = make(chan *spotify.Client)
	state = "music-utils"
)

func authFlow() (*Service, error) {
	// Ensure Spotify application ID and secret are set
	if viper.GetString("spotify.client_id") == "" || viper.GetString("spotify.client_secret") == "" {
		log.Fatal().Msg("Spotify client ID and/or secret not set")
	}

	// Check if Spotify access and refresh token is set
	// If set, fetch and return client
	if viper.GetString("spotify.access_token") == "" || viper.GetString("spotify.refresh_token") == "" {
		log.Warn().Msg("Spotify access token and refresh token not set")
		client, err := auth()
		if err != nil {
			return nil, fmt.Errorf("error authenticating with Spotify: %w", err)
		}

		return &Service{client: client}, nil
	}

	// Continue with auth flow
	// Use Spotify refresh token to get a new token and create client
	tok := &oauth2.Token{
		AccessToken:  viper.GetString("spotify.access_token"),
		RefreshToken: viper.GetString("spotify.refresh_token"),
		Expiry:       viper.GetTime("spotify.expiry"),
		TokenType:    viper.GetString("spotify.token_type"),
	}
	spotClientID := viper.GetString("spotify.client_id")
	spotClientSecret := viper.GetString("spotify.client_secret")
	redirectURI := viper.GetString("spotify.redirect_uri")
	auth := spotifyauth.New(spotifyauth.WithClientID(spotClientID), spotifyauth.WithClientSecret(spotClientSecret), spotifyauth.WithRedirectURL(redirectURI), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopePlaylistReadPrivate))

	client := spotify.New(auth.Client(context.Background(), tok))

	newTok, _ := client.Token()
	viper.Set("spotify.access_token", newTok.AccessToken)
	viper.Set("spotify.expiry", newTok.Expiry)
	viper.Set("spotify.token_type", newTok.TokenType)
	err := viper.WriteConfig()
	if err != nil {
		return nil, fmt.Errorf("error writing config: %w", err)
	}

	user, err := client.CurrentUser(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting current user: %w", err)
	}
	log.Info().Msgf("Spotify - logged in as: %s", user.ID)

	return &Service{client: client}, nil
}

func auth() (*spotify.Client, error) {
	spotClientID := viper.GetString("spotify.client_id")
	spotClientSecret := viper.GetString("spotify.client_secret")
	redirectURI := viper.GetString("spotify.redirect_uri")
	auth := spotifyauth.New(spotifyauth.WithClientID(spotClientID), spotifyauth.WithClientSecret(spotClientSecret), spotifyauth.WithRedirectURL(redirectURI), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopePlaylistReadPrivate))
	// Start an HTTP server
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	go func() {
		err := http.ListenAndServe(":28542", nil)
		if err != nil {
			log.Error().Msgf("Error starting HTTP server: %w", err)
		}
	}()

	url := auth.AuthURL(state)
	log.Info().Msgf("Please log in to Spotify by visiting the following page in your browser: %s", url)

	// wait for auth to complete
	client := <-ch

	// use the client to make calls that require authorization
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting current user: %w", err)
	}
	log.Info().Msgf("Spotify - logged in as: %s", user.ID)
	return client, nil
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	spotClientID := viper.GetString("spotify.client_id")
	spotClientSecret := viper.GetString("spotify.client_secret")
	redirectURI := viper.GetString("spotify.redirect_uri")
	auth := spotifyauth.New(spotifyauth.WithClientID(spotClientID), spotifyauth.WithClientSecret(spotClientSecret), spotifyauth.WithRedirectURL(redirectURI), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopePlaylistReadPrivate))

	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Error().Msgf("Couldn't get token: %w", err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Error().Msgf("State mismatch: %s != %s\n", st, state)
	}

	// Save token to config
	viper.Set("spotify.access_token", tok.AccessToken)
	viper.Set("spotify.refresh_token", tok.RefreshToken)
	viper.Set("spotify.expiry", tok.Expiry)
	viper.Set("spotify.token_type", tok.TokenType)
	err = viper.WriteConfig()
	if err != nil {
		log.Error().Msgf("Error writing config file: %w", err)
	}

	// use the token to get an authenticated client
	client := spotify.New(auth.Client(r.Context(), tok))
	ch <- client
}
