// main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ahmetardacelik/SpotifyStats/spotify" // Adjust the import path as needed
)

var spotifyClient spotify.Client

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// Redirect user to Spotify's authorization page
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

	// Initialize Spotify client with the obtained token
	spotifyClient.Initialize(token)

	// Redirect to the top tracks endpoint
	http.Redirect(w, r, "/top-tracks", http.StatusFound)
}

func main() {
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	spotifyClient = spotify.Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/callback", callbackHandler)

	http.HandleFunc("/top-artists", func(w http.ResponseWriter, r *http.Request) {
		data, err := spotifyClient.FetchTopArtists()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	http.HandleFunc("/top-tracks", func(w http.ResponseWriter, r *http.Request) {
		data, err := spotifyClient.FetchTopTracks()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	fmt.Println("Server is running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
