package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/ahmetardacelik/fromMac/repository"
	"github.com/ahmetardacelik/fromMac/spotify"
	"github.com/gorilla/mux"
)

var dbConn *sql.DB

func main() {
	var err error
	dbConn, err = repository.InitializeDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbConn.Close()

	spotifyRepository := repository.NewSpotifyRepository(dbConn)
	spotifyService := spotify.NewSpotifyService(spotifyRepository)

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	spotifyClient := spotify.Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Repository:   spotifyRepository,
	}

	spotifyHandler := NewHandler(spotifyService, spotifyClient)

	// HTTP router setup
	router := mux.NewRouter()
	spotifyHandler.RegisterRoutes(router)

	// Start the periodic data fetching
	go spotifyClient.PeriodicallyFetchData()
	

	log.Println("HTTP server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", router))
}
