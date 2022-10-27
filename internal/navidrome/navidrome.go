package navidrome

import (
	"github.com/rs/zerolog/log"
	"github.com/zibbp/music-utils/internal/database"
)

type Service struct {
	Db *database.Database
}

func InitializeService() (*Service, error) {
	// Setup database
	db, err := database.Setup()
	if err != nil {
		log.Fatal().Msgf("Error initializing database: %w", err)
	}
	return &Service{
		Db: db,
	}, nil
}
