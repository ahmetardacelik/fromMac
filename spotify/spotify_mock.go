// spotify_mock.go
package spotify

import (
	"context"
	"golang.org/x/oauth2"
)

type MockOAuth2Config struct {
	AuthURL  string
	TokenURL string
	Token    *oauth2.Token
	Err      error
}

func (m *MockOAuth2Config) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return m.AuthURL
}

func (m *MockOAuth2Config) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return m.Token, m.Err
}
