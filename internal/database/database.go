package database

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

type Database struct {
	DB *sql.DB
}

func Setup() (*Database, error) {
	log.Info().Msg("Opening Navidrome database connection")
	db, err := sql.Open("sqlite3", "/navidrome/navidrome.db")
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	return &Database{DB: db}, nil
}

func (d *Database) FindTrack(title, artist string) (string, error) {
	var path string
	err := d.DB.QueryRow("SELECT path FROM media_file WHERE title LIKE ? AND artist LIKE ?", "%"+title+"%", "%"+artist+"%").Scan(&path)
	if err != nil {
		return "", fmt.Errorf("error finding track: %w", err)
	}
	return path, nil
}
