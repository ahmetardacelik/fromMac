// spotify/models.go
package spotify

import (
    "encoding/json"
)

// Artist represents an artist in the Spotify response
type Artist struct {
    ExternalURLs struct {
        Spotify string `json:"spotify"`
    } `json:"external_urls"`
    Followers struct {
        Total int `json:"total"`
    } `json:"followers"`
    Genres  []string `json:"genres"`
    Href    string   `json:"href"`
    ID      string   `json:"id"`
    Images  []struct {
        URL    string `json:"url"`
        Height int    `json:"height"`
        Width  int    `json:"width"`
    } `json:"images"`
    Name      string `json:"name"`
    Popularity int    `json:"popularity"`
    Type      string `json:"type"`
    URI       string `json:"uri"`
}

// TopArtistsResponse represents the response from the top artists endpoint
type TopArtistsResponse struct {
    Items    []Artist `json:"items"`
    Total    int      `json:"total"`
    Limit    int      `json:"limit"`
    Offset   int      `json:"offset"`
    Href     string   `json:"href"`
    Next     string   `json:"next"`
    Previous string   `json:"previous"`
}

// UnmarshalTopArtists unmarshals the JSON response into a TopArtistsResponse struct
func UnmarshalTopArtists(data []byte) (TopArtistsResponse, error) {
    var r TopArtistsResponse
    err := json.Unmarshal(data, &r)
    return r, err
}
