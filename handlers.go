package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/ahmetardacelik/fromMac/models"
	"github.com/ahmetardacelik/fromMac/repository"
	"github.com/ahmetardacelik/fromMac/spotify"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

type Handler struct {
	SpotifyService *spotify.SpotifyService
	Client         spotify.Client
}

func NewHandler(service *spotify.SpotifyService, cl spotify.Client) *Handler {
	return &Handler{
		SpotifyService: service,
		Client:         cl,
	}
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/login", h.login).Methods("GET")
	router.HandleFunc("/callback", h.callback).Methods("GET")
	router.HandleFunc("/top-artists", h.topArtistsHandler).Methods("GET")
	router.HandleFunc("/analyze", h.analyzeHandler).Methods("GET")
	router.HandleFunc("/fetch-data", h.fetchRecordedDataHandler).Methods("GET")
	router.HandleFunc("/", serveIndex).Methods("GET")
}

func (h *Handler) topArtistsHandler(w http.ResponseWriter, r *http.Request) {
	topArtists, err := h.Client.FetchTopArtistsWithParsing()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var artists []repository.Artist
	var genres [][]string
	for _, artist := range topArtists.Items {
		artists = append(artists, convertToRepositoryArtist(artist))
		genres = append(genres, artist.Genres)
	}

	err = h.Client.Repository.InsertData(h.Client.UserID, artists, genres)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	genreCount := make(map[string]int)
	for _, artist := range topArtists.Items {
		for _, genre := range artist.Genres {
			genreCount[genre]++
		}
	}

	type genre struct {
		Name  string
		Count int
	}
	var genresSlice []genre
	for name, count := range genreCount {
		genresSlice = append(genresSlice, genre{Name: name, Count: count})
	}
	sort.Slice(genresSlice, func(i, j int) bool {
		return genresSlice[i].Count > genresSlice[j].Count
	})

	response := struct {
		Artists []repository.Artist `json:"artists"`
		Genres  []genre             `json:"genres"`
	}{
		Artists: artists,
		Genres:  genresSlice,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	url := spotify.Config.AuthCodeURL("", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
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

	err = h.Client.Initialize(token)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to initialize Spotify client: %v", err), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/top-artists", http.StatusFound)
}
func convertToRepositoryArtist(artist models.Artist) repository.Artist {
	return repository.Artist{
		ID:         artist.ID,
		Name:       artist.Name,
		Popularity: artist.Popularity,
		Followers:  artist.Followers.Total,
	}
}

func (h *Handler) analyzeData() {
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

func (h *Handler) analyzeHandler(w http.ResponseWriter, r *http.Request) {
	h.analyzeData()
	w.Write([]byte("Analysis complete. Check server logs for details."))
}

func (h *Handler) fetchRecordedDataHandler(w http.ResponseWriter, r *http.Request) {
	artists, err := repository.FetchArtistsData(dbConn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch artists data: %v", err), http.StatusInternalServerError)
		return
	}

	genres, err := repository.FetchGenresData(dbConn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch genres data: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		Artists []repository.Artist `json:"artists"`
		Genres  map[string]int      `json:"genres"`
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
