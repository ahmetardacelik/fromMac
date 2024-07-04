package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
	"github.com/ahmetardacelik/fromMac/spotify"
)

func init() {
	spotify.Config = &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Scopes:       []string{"user-top-read"},
		RedirectURL:  "http://localhost:8080/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
	}
}

func TestLoginHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/login", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(loginHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusTemporaryRedirect {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusTemporaryRedirect)
	}

	expected := spotify.Config.AuthCodeURL("", oauth2.AccessTypeOffline)
	if location := rr.Header().Get("Location"); location != expected {
		t.Errorf("handler returned unexpected Location header: got %v want %v",
			location, expected)
	}
}
