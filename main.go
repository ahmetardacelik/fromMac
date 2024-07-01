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
	spotifyClient       spotify.Client
	dbConn              *sql.DB
	comparisonDbConn    *sql.DB
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

func insertData(artists []spotify.Artist, dbConn *sql.DB) error {
	tx, err := dbConn.Begin()
	if err != nil {
		return err
	}

	for _, artist := range artists {
		_, err = tx.Exec("INSERT OR REPLACE INTO artists (id, name, popularity, followers, timestamp) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)",
			artist.ID, artist.Name, artist.Popularity, artist.Followers.Total)
		if err != nil {
			tx.Rollback()
			return err
		}

		for _, genre := range artist.Genres {
			_, err = tx.Exec("INSERT INTO genres (artist_id, genre, timestamp) VALUES (?, ?, CURRENT_TIMESTAMP)",
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

	err = insertData(topArtists.Items, dbConn)
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

		err = insertData(topArtists.Items, dbConn)
		if err != nil {
			log.Printf("Error inserting data into main database: %v", err)
		}

		err = insertData(topArtists.Items, comparisonDbConn)
		if err != nil {
			log.Printf("Error inserting data into comparison database: %v", err)
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

func fetchGenresData(dbConn *sql.DB) (map[string]int, error) {
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

func compareDataHandler(w http.ResponseWriter, r *http.Request) {
	mainGenres, err := fetchGenresData(dbConn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch genres from main database: %v", err), http.StatusInternalServerError)
		return
	}

	comparisonGenres, err := fetchGenresData(comparisonDbConn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch genres from comparison database: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		MainGenres       map[string]int `json:"main_genres"`
		ComparisonGenres map[string]int `json:"comparison_genres"`
	}{
		MainGenres:       mainGenres,
		ComparisonGenres: comparisonGenres,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

	comparisonDbConn, err = db.InitializeComparisonDB()
	if err != nil {
		log.Fatalf("Failed to initialize comparison database: %v", err)
	}
	defer comparisonDbConn.Close()

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
	http.HandleFunc("/fetch-data", fetchRecordedDataHandler)
	http.HandleFunc("/compare-data", compareDataHandler) // New endpoint

	fmt.Println("Server is running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
func fetchRecordedDataHandler(w http.ResponseWriter, r *http.Request) {
	artists, err := fetchArtistsData(dbConn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch artists data: %v", err), http.StatusInternalServerError)
		return
	}

	genres, err := fetchGenresData(dbConn)
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

func fetchArtistsData(dbConn *sql.DB) ([]spotify.Artist, error) {
	rows, err := dbConn.Query("SELECT id, name, popularity, followers, timestamp FROM artists")
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
