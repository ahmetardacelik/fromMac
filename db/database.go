// db/db.go
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

	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT
	);`

	createArtistsTable := `
	CREATE TABLE IF NOT EXISTS artists (
		id TEXT PRIMARY KEY,
		name TEXT,
		popularity INTEGER,
		followers INTEGER
	);`

	createGenresTable := `
	CREATE TABLE IF NOT EXISTS genres (
		artist_id TEXT,
		genre TEXT,
		FOREIGN KEY (artist_id) REFERENCES artists(id)
	);`

	createUserArtistsTable := `
	CREATE TABLE IF NOT EXISTS user_artists (
		user_id TEXT,
		artist_id TEXT,
		rank INTEGER,
		timestamp DATETIME DEFAULT (DATETIME(CURRENT_TIMESTAMP, 'LOCALTIME')),
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (artist_id) REFERENCES artists(id)
	);`

	_, err = db.Exec(createUsersTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(createArtistsTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(createGenresTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(createUserArtistsTable)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// InsertUser inserts a user into the users table
func InsertUser(dbConn *sql.DB, userID, username string) error {
	_, err := dbConn.Exec("INSERT OR IGNORE INTO users (id, username) VALUES (?, ?)", userID, username)
	return err
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
