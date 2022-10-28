package database

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
	"strings"
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

func (d *Database) FindTrack(title, album, artist string) (string, error) {
	var path string
	// Attempt to find track by title, album and artist and then mutating the strings
	// This is not 100% accurate, but it's the best we can do
	// View missing json tracks to manually import missed ones
	err := d.DB.QueryRow("SELECT path FROM media_file WHERE title LIKE ? AND artist LIKE ?", "%"+title+"%", "%"+artist+"%").Scan(&path)
	if err != nil {
		// attempt to find track by album
		err = d.DB.QueryRow("SELECT path FROM media_file WHERE title LIKE ? AND album LIKE ?", "%"+title+"%", "%"+album+"%").Scan(&path)
		if err != nil {
			// Cleanup strings
			// Get title before first parenthesis
			title = strings.Split(title, " (")[0]
			// Replace ’ with '
			title = strings.ReplaceAll(title, "’", "'")
			artist = strings.ReplaceAll(artist, "’", "'")
			err := d.DB.QueryRow("SELECT path FROM media_file WHERE title LIKE ? AND artist LIKE ?", "%"+title+"%", "%"+artist+"%").Scan(&path)
			if err != nil {
				// Rplace ' with ’
				title = strings.ReplaceAll(title, "'", "’")
				artist = strings.ReplaceAll(artist, "'", "’")
				err := d.DB.QueryRow("SELECT path FROM media_file WHERE title LIKE ? AND artist LIKE ?", "%"+title+"%", "%"+artist+"%").Scan(&path)
				if err != nil {
					return "", fmt.Errorf("error finding track: %w", err)
				}
			}
		}
	}
	return path, nil
}
