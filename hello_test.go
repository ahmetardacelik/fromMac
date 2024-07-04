// main_test.go
package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ahmetardacelik/fromMac/spotify"
	"golang.org/x/oauth2"
)

func TestLoginHandler(t *testing.T) {
	mockConfig := &spotify.MockOAuth2Config{
		AuthURL: "https://accounts.spotify.com/authorize?client_id=test-client-id&response_type=code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcallback&scope=user-top-read",
	}

	req, err := http.NewRequest("GET", "/login", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := loginHandler(mockConfig)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusTemporaryRedirect {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusTemporaryRedirect)
	}

	expected := mockConfig.AuthCodeURL("", oauth2.AccessTypeOffline)
	if location := rr.Header().Get("Location"); location != expected {
		t.Errorf("handler returned unexpected Location header: got %v want %v", location, expected)
	}
}

func TestCallbackHandler(t *testing.T) {
	mockConfig := &spotify.MockOAuth2Config{
		Token: &oauth2.Token{AccessToken: "mock-access-token"},
	}

	t.Run("Test Token Exchange", func(t *testing.T) {
		token, err := mockConfig.Exchange(context.Background(), "test-code")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if token.AccessToken != "mock-access-token" {
			t.Fatalf("Expected access token 'mock-access-token', got %v", token.AccessToken)
		}
	})

	t.Run("Test Callback Handler", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/callback?code=test-code", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := callbackHandler(mockConfig)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusFound)
		}

		expectedLocation := "/top-artists"
		if location := rr.Header().Get("Location"); location != expectedLocation {
			t.Errorf("handler returned wrong Location header: got %v want %v", location, expectedLocation)
		}
	})
}
