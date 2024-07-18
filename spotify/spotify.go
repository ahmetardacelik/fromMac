package spotify

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"


	"github.com/ahmetardacelik/fromMac/models"
	"golang.org/x/oauth2"
)

type Client struct {
	ClientID     string
	ClientSecret string
	Token        *oauth2.Token
	Client       *http.Client
	UserID       string
	Username     string
}
type SpotifyService struct {
	SpotifyRepository SpotifyRepository

}
type Handler struct {
	SpotifyRepository SpotifyRepository
	Client Client
}



func (c *Client) Initialize(dbConn *sql.DB, token *oauth2.Token) error {
	c.Token = token
	c.Client = oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(token))

	// Fetch user profile
	profile, err := c.fetchUserProfile()
	if err != nil {
		return err
	}
	c.UserID = profile.ID
	c.Username = profile.DisplayName

	// Insert user into the database
	err = db.InsertUser(dbConn, c.UserID, c.Username)
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
