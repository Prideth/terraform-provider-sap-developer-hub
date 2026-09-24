// Package auth builds an OAuth2 client-credentials authenticated HTTP
// client for the Developer Hub "devportal-apiaccess" service plan
// (see DESIGN.md §4), with a token cache that can be explicitly invalidated
// after a 401 so the retrying HTTP client can fetch a fresh token instead
// of retrying with a token already known to be bad.
package auth

import (
	"context"
	"net/http"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Config holds the OAuth2 client-credentials parameters from a Developer
// Hub "devportal-apiaccess" service key.
type Config struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
}

// invalidatableTokenSource caches a token until it expires or is explicitly
// invalidated. golang.org/x/oauth2's own ReuseTokenSource has no exported
// invalidate hook, which is why this is hand-rolled rather than using it
// directly.
type invalidatableTokenSource struct {
	mu     sync.Mutex
	fetch  func() (*oauth2.Token, error)
	cached *oauth2.Token
}

func (s *invalidatableTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cached != nil && s.cached.Valid() {
		return s.cached, nil
	}

	tok, err := s.fetch()
	if err != nil {
		return nil, err
	}
	s.cached = tok
	return tok, nil
}

func (s *invalidatableTokenSource) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached = nil
}

// HTTPClient builds an *http.Client that transparently attaches and
// refreshes a bearer token, and returns an invalidate function the caller
// can invoke after receiving a 401 so the next request fetches a fresh
// token rather than retrying with the same bad one.
func (c Config) HTTPClient(ctx context.Context, base *http.Client) (*http.Client, func(), error) {
	ccConfig := clientcredentials.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		TokenURL:     c.TokenURL,
		AuthStyle:    oauth2.AuthStyleInHeader,
	}

	baseCtx := context.WithValue(ctx, oauth2.HTTPClient, base)
	source := &invalidatableTokenSource{
		fetch: func() (*oauth2.Token, error) {
			return ccConfig.Token(baseCtx)
		},
	}

	client := &http.Client{
		Transport: &oauth2.Transport{
			Source: source,
			Base:   base.Transport,
		},
	}

	return client, source.Invalidate, nil
}
