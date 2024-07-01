// main.go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "sort"

    "github.com/ahmetardacelik/fromMac/spotify"
    "golang.org/x/oauth2"
)

var spotifyClient spotify.Client

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

func topArtistsHandler(w http.ResponseWriter, r *http.Request) {
    topArtists, err := spotifyClient.FetchTopArtistsWithParsing()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Print artist names in order
    fmt.Println("Top Artists:")
    for _, artist := range topArtists.Items {
        fmt.Println(artist.Name)
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

    // Print most listened genres
    fmt.Println("\nMost Listened Genres:")
    for _, g := range genres {
        fmt.Printf("%s: %d\n", g.Name, g.Count)
    }
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
    http.HandleFunc("/top-artists", topArtistsHandler)

    fmt.Println("Server is running at http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
