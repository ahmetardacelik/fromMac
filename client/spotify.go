// spotify/spotify.go
package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/oauth2"
)

// Client struct to hold OAuth2 config and access token
type Client struct {
	ClientID     string
	ClientSecret string
	Token        *oauth2.Token
	Client       *http.Client
}

// Initialize method to set up the Spotify client with the access token
func (c *Client) Initialize(token *oauth2.Token) {
	c.Token = token
	c.Client = oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(token))
}

// FetchTopArtists method to fetch top artists
func (c *Client) FetchTopArtists() ([]byte, error) {
	return c.makeRequest("https://api.spotify.com/v1/me/top/artists")
}

// FetchTopTracks method to fetch top tracks
func (c *Client) FetchTopTracks() ([]byte, error) {
	return c.makeRequest("https://api.spotify.com/v1/me/top/tracks")
}

// makeRequest method to make a request to Spotify API
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

	// Print beautified JSON response to the console
	fmt.Println("Beautified Response from Spotify:")
	fmt.Println(prettyJSON.String())

	return prettyJSON.Bytes(), nil
}

// Config for OAuth2 configuration
var Config = &oauth2.Config{
	ClientID:     os.Getenv("CLIENT_ID"), // Ensure these are set in your environment
	ClientSecret: os.Getenv("CLIENT_SECRET"),
	Scopes:       []string{"user-top-read"},
	RedirectURL:  "http://localhost:8080/callback",
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://accounts.spotify.com/authorize",
		TokenURL: "https://accounts.spotify.com/api/token",
	},
}
