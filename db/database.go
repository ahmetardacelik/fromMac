package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3" // Import the SQLite3 driver
	"github.com/ahmetardacelik/fromMac/spotify"
)

// InitializeDB initializes the database and creates the necessary tables
func InitializeDB() (*sql.DB, error) {
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

// InsertData inserts artist data into the database
func InsertData(dbConn *sql.DB, artists []spotify.Artist) error {
	tx, err := dbConn.Begin()
	if err != nil {
		return err
	}

	for _, artist := range artists {
		_, err = tx.Exec("INSERT OR REPLACE INTO artists (id, name, popularity, followers) VALUES (?, ?, ?, ?)",
			artist.ID, artist.Name, artist.Popularity, artist.Followers.Total)
		if err != nil {
			tx.Rollback()
			return err
		}

		for _, genre := range artist.Genres {
			_, err = tx.Exec("INSERT INTO genres (artist_id, genre) VALUES (?, ?)",
				artist.ID, genre)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

// FetchGenresData fetches genre data from the database
func FetchGenresData(dbConn *sql.DB) (map[string]int, error) {
	rows, err := dbConn.Query("SELECT genre, COUNT(genre) as count FROM genres GROUP BY genre")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	genres := make(map[string]int)
	for rows.Next() {
		var genre string
		var count int
		err := rows.Scan(&genre, &count)
		if err != nil {
			return nil, err
		}
		genres[genre] = count
	}
	return genres, nil
}

// FetchArtistsData fetches artist data from the database
func FetchArtistsData(dbConn *sql.DB) ([]spotify.Artist, error) {
	rows, err := dbConn.Query("SELECT id, name, popularity, followers FROM artists")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []spotify.Artist
	for rows.Next() {
		var artist spotify.Artist
		var followers int
		err := rows.Scan(&artist.ID, &artist.Name, &artist.Popularity, &followers)
		if err != nil {
			return nil, err
		}
		artist.Followers.Total = followers
		artists = append(artists, artist)
	}
	return artists, nil
}
