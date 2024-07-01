package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/ahmetardacelik/fromMac/db"
	"github.com/ahmetardacelik/fromMac/spotify"
)

// InsertData inserts artist data into the database
func insertData(dbConn *sql.DB, artists []spotify.Artist) error {
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

func topArtistsHandler(w http.ResponseWriter, r *http.Request) {
	topArtists, err := spotifyClient.FetchTopArtistsWithParsing()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = insertData(dbConn, topArtists.Items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate most listened genres
	genreCount := make(map[string]int)
	for _, artist := range topArtists.Items {
		for _, genre := range artist.Genres {
			genreCount[genre]++
		}
	}

	// Create a sorted slice of genres by count
	type genre struct {
		Name  string
		Count int
	}
	var genres []genre
	for name, count := range genreCount {
		genres = append(genres, genre{Name: name, Count: count})
	}
	sort.Slice(genres, func(i, j int) bool {
		return genres[i].Count > genres[j].Count
	})

	// Create a response struct to send JSON data
	response := struct {
		Artists []spotify.Artist `json:"artists"`
		Genres  []genre          `json:"genres"`
	}{
		Artists: topArtists.Items,
		Genres:  genres,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func periodicallyFetchData() {
	for {
		if spotifyClient.Client == nil {
			log.Println("Spotify client not initialized yet")
			time.Sleep(1 * time.Minute)
			continue
		}

		topArtists, err := spotifyClient.FetchTopArtistsWithParsing()
		if err != nil {
			log.Printf("Error fetching top artists: %v", err)
			continue
		}

		err = insertData(dbConn, topArtists.Items)
		if err != nil {
			log.Printf("Error inserting data: %v", err)
		}

		// Sleep for an hour before fetching the data again
		time.Sleep(1 * time.Minute)
	}
}

func analyzeData() {
	rows, err := dbConn.Query(`
		SELECT genre, COUNT(genre) as count
		FROM genres
		WHERE timestamp >= datetime('now', '-7 days')
		GROUP BY genre
		ORDER BY count DESC
	`)
	if err != nil {
		log.Fatalf("Failed to analyze data: %v", err)
	}
	defer rows.Close()

	fmt.Println("Genres listened to in the last 7 days:")
	for rows.Next() {
		var genre string
		var count int
		err = rows.Scan(&genre, &count)
		if err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		fmt.Printf("%s: %d\n", genre, count)
	}
}

func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	analyzeData()
	w.Write([]byte("Analysis complete. Check server logs for details."))
}

func fetchRecordedDataHandler(w http.ResponseWriter, r *http.Request) {
	artists, err := db.FetchArtistsData(dbConn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch artists data: %v", err), http.StatusInternalServerError)
		return
	}

	genres, err := db.FetchGenresData(dbConn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch genres data: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		Artists []spotify.Artist `json:"artists"`
		Genres  map[string]int   `json:"genres"`
	}{
		Artists: artists,
		Genres:  genres,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}
