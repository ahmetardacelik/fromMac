package repository

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SpotifyRepository struct {
	DB *sql.DB
}

type Repository interface {
	InsertUser(userID, username string) error
	InsertData(userID string, artists []Artist, genres [][]string) error
}

func NewSpotifyRepository(db *sql.DB) *SpotifyRepository {
	return &SpotifyRepository{
		DB: db,
	}
}

// Ensure SpotifyRepository implements Repository
var _ Repository = &SpotifyRepository{}

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
		timestamp DATETIME,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (artist_id) REFERENCES artists(id),
		PRIMARY KEY (user_id, artist_id)
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

func (r *SpotifyRepository) InsertUser(userID, username string) error {
	_, err := r.DB.Exec("INSERT OR IGNORE INTO users (id, username) VALUES (?, ?)", userID, username)
	return err
}

func (r *SpotifyRepository) InsertData(userID string, artists []Artist, genres [][]string) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	rank := 1
	location, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		return err
	}
	timestamp := time.Now().In(location).Format("2006-01-02 15:04:05")

	for i, artist := range artists {
		_, err = tx.Exec("INSERT OR REPLACE INTO artists (id, name, popularity, followers) VALUES (?, ?, ?, ?)",
			artist.ID, artist.Name, artist.Popularity, artist.Followers)
		if err != nil {
			tx.Rollback()
			return err
		}

		for _, genre := range genres[i] {
			_, err = tx.Exec("INSERT OR IGNORE INTO genres (artist_id, genre) VALUES (?, ?)",
				artist.ID, genre)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

		_, err = tx.Exec("INSERT OR REPLACE INTO user_artists (user_id, artist_id, rank, timestamp) VALUES (?, ?, ?, ?)",
			userID, artist.ID, rank, timestamp)
		if err != nil {
			tx.Rollback()
			return err
		}

		rank++
	}

	return tx.Commit()
}

type Artist struct {
	ID         string
	Name       string
	Popularity int
	Followers  int
}

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

func FetchArtistsData(dbConn *sql.DB) ([]Artist, error) {
	rows, err := dbConn.Query("SELECT id, name, popularity, followers FROM artists")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []Artist
	for rows.Next() {
		var artist Artist
		err := rows.Scan(&artist.ID, &artist.Name, &artist.Popularity, &artist.Followers)
		if err != nil {
			return nil, err
		}
		artists = append(artists, artist)
	}
	return artists, nil
}
