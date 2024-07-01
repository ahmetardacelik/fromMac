// main.go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/ahmetardacelik/fromMac/db"
	"github.com/ahmetardacelik/fromMac/spotify"
	"golang.org/x/oauth2"
)

var (
	spotifyClient spotify.Client
	dbConn        *sql.DB
)

func loginHandler(w http.ResponseWriter, r *http.Request) {
	url := spotify.Config.AuthCodeURL("", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not provided", http.StatusBadRequest)
		return
	}
	token, err := spotify.Config.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange token: %v", err), http.StatusInternalServerError)
		return
	}

	spotifyClient.Initialize(token)
	http.Redirect(w, r, "/top-artists", http.StatusFound)
}

func insertData(artists []spotify.Artist) error {
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

	err = insertData(topArtists.Items)
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
		topArtists, err := spotifyClient.FetchTopArtistsWithParsing()
		if err != nil {
			log.Printf("Error fetching top artists: %v", err)
			continue
		}

		err = insertData(topArtists.Items)
		if err != nil {
			log.Printf("Error inserting data: %v", err)
		}

		// Sleep for an hour before fetching the data again
		time.Sleep(1 * time.Hour)
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

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func main() {
	var err error
	dbConn, err = db.InitializeDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbConn.Close()

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	spotifyClient = spotify.Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	// Start the periodic data fetching
	go periodicallyFetchData()

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/callback", callbackHandler)
	http.HandleFunc("/top-artists", topArtistsHandler)
	http.HandleFunc("/analyze", analyzeHandler)
	http.HandleFunc("/", serveIndex)

	fmt.Println("Server is running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
