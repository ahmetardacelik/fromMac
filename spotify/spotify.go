package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ahmetardacelik/fromMac/models"
	"github.com/ahmetardacelik/fromMac/repository"
	"golang.org/x/oauth2"
)

type Client struct {
	ClientID     string
	ClientSecret string
	Token        *oauth2.Token
	Client       *http.Client
	UserID       string
	Username     string
	Repository   repository.Repository
}

type SpotifyService struct {
	SpotifyRepository repository.Repository
}

func NewSpotifyService(repository repository.Repository) *SpotifyService {
	return &SpotifyService{
		SpotifyRepository: repository,
	}
}

func (c *Client) Initialize(token *oauth2.Token) error {
	c.Token = token
	c.Client = oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(token))

	profile, err := c.fetchUserProfile()
	if err != nil {
		return err
	}
	c.UserID = profile.ID
	c.Username = profile.DisplayName

	err = c.Repository.InsertUser(c.UserID, c.Username)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) fetchUserProfile() (UserProfile, error) {
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me", nil)
	if err != nil {
		return UserProfile{}, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token.AccessToken))
	resp, err := c.Client.Do(req)
	if err != nil {
		return UserProfile{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UserProfile{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UserProfile{}, err
	}

	var profile UserProfile
	err = json.Unmarshal(body, &profile)
	if err != nil {
		return UserProfile{}, err
	}

	return profile, nil
}

type UserProfile struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

func (c *Client) FetchTopArtists() ([]byte, error) {
	return c.makeRequest("https://api.spotify.com/v1/me/top/artists")
}

func (c *Client) FetchTopTracks() ([]byte, error) {
	return c.makeRequest("https://api.spotify.com/v1/me/top/tracks")
}

func (c *Client) makeRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token.AccessToken))
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error on request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, body, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error beautifying JSON response: %w", err)
	}

	fmt.Println("Beautified Response from Spotify:")
	fmt.Println(prettyJSON.String())

	return prettyJSON.Bytes(), nil
}

func (c *Client) FetchTopArtistsWithParsing() (models.TopArtistsResponse, error) {
	data, err := c.makeRequest("https://api.spotify.com/v1/me/top/artists")
	if err != nil {
		return models.TopArtistsResponse{}, err
	}

	return models.UnmarshalTopArtists(data)
}

var Config = &oauth2.Config{
	ClientID:     os.Getenv("CLIENT_ID"),
	ClientSecret: os.Getenv("CLIENT_SECRET"),
	Scopes:       []string{"user-top-read", "user-read-private"},
	RedirectURL:  "http://localhost:8080/callback",
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://accounts.spotify.com/authorize",
		TokenURL: "https://accounts.spotify.com/api/token",
	},
}
func convertToRepositoryArtist(artist models.Artist) repository.Artist {
	return repository.Artist{
		ID:         artist.ID,
		Name:       artist.Name,
		Popularity: artist.Popularity,
		Followers:  artist.Followers.Total,
	}
}

func (c *Client) PeriodicallyFetchData() {
	for {
		if c == nil {
			log.Println("Spotify client not initialized yet")
			time.Sleep(1 * time.Minute)
			continue
		}

		topArtists, err := c.FetchTopArtistsWithParsing()
		if err != nil {
			log.Printf("Error fetching top artists: %v", err)
			continue
		}

		var artists []repository.Artist
		var genres [][]string
		for _, artist := range topArtists.Items {
			artists = append(artists, convertToRepositoryArtist(artist))
			genres = append(genres, artist.Genres)
		}

		err = c.Repository.InsertData(c.UserID, artists, genres)
		if err != nil {
			log.Printf("Error inserting data: %v", err)
		}

		time.Sleep(1 * time.Hour)
	}
}
