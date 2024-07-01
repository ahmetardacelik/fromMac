// database.go
package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func initializeDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./spotify_data.db")
	if err != nil {
		return nil, err
	}

	createArtistsTable := `
	CREATE TABLE IF NOT EXISTS artists (
		id TEXT PRIMARY KEY,
		name TEXT,
		popularity INTEGER,
		followers INTEGER,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createGenresTable := `
	CREATE TABLE IF NOT EXISTS genres (
		artist_id TEXT,
		genre TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (artist_id) REFERENCES artists(id)
	);`

	_, err = db.Exec(createArtistsTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(createGenresTable)
	if err != nil {
		return nil, err
	}

	return db, nil
}
