// main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

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
	http.HandleFunc("/fetch-data", fetchRecordedDataHandler)
	http.HandleFunc("/", serveIndex)

	fmt.Println("Server is running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
