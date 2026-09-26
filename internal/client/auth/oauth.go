// Package auth builds an OAuth2 client-credentials authenticated HTTP
// client for the Developer Hub "devportal-apiaccess" service plan
// (see DESIGN.md §4), with a token cache that can be explicitly invalidated
// after a 401 so the retrying HTTP client can fetch a fresh token instead
// of retrying with a token already known to be bad.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// maxTokenResponseBytes bounds how much of a token endpoint response is
// ever read into memory or surfaced in an error message.
const maxTokenResponseBytes = 1 << 20 // 1 MiB

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
	// Token fetches happen lazily, long after HTTPClient returns: Terraform
	// cancels the context it passes to ConfigureProvider once configuration
	// is done, so the token requests keep its values but not its
	// cancellation.
	baseCtx := context.WithoutCancel(ctx)
	source := &invalidatableTokenSource{
		fetch: func() (*oauth2.Token, error) {
			return fetchToken(baseCtx, base, c.TokenURL, c.ClientID, c.ClientSecret)
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

// tokenResponse is the RFC 6749 §5.1 success body a client-credentials
// grant returns.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// tokenErrorResponse is the RFC 6749 §5.2 error body a client-credentials
// grant returns.
type tokenErrorResponse struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorURI         string `json:"error_uri"`
}

// fetchToken performs the client-credentials grant by hand rather than
// through golang.org/x/oauth2/clientcredentials. That package's
// AuthStyleInHeader percent-encodes the client ID and secret (per RFC
// 6749 §2.3.1) before base64-encoding them into the Basic auth header. SAP
// XSUAA - the token endpoint every "devportal-apiaccess" service key points
// to - does not decode that encoding back out, so a client_id or secret
// containing "!", "|", "$" or "=" (routine in an XSUAA service key) is
// rejected as "invalid_client: Bad credentials" even though the raw
// credentials are correct. net/http's Request.SetBasicAuth base64-encodes
// the raw bytes with no percent-encoding, matching what XSUAA expects.
func fetchToken(ctx context.Context, base *http.Client, tokenURL, clientID, clientSecret string) (*oauth2.Token, error) {
	body := strings.NewReader(url.Values{"grant_type": {"client_credentials"}}.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, body)
	if err != nil {
		return nil, fmt.Errorf("building the OAuth token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := base.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling the OAuth token endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxTokenResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading the OAuth token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		retrieveErr := &oauth2.RetrieveError{Response: resp, Body: respBody}
		var errBody tokenErrorResponse
		if json.Unmarshal(respBody, &errBody) == nil {
			retrieveErr.ErrorCode = errBody.ErrorCode
			retrieveErr.ErrorDescription = errBody.ErrorDescription
			retrieveErr.ErrorURI = errBody.ErrorURI
		}
		return nil, retrieveErr
	}

	var tok tokenResponse
	if err := json.Unmarshal(respBody, &tok); err != nil {
		return nil, fmt.Errorf("parsing the OAuth token response: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("the OAuth token endpoint returned no access_token")
	}

	tokenType := tok.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	token := &oauth2.Token{
		AccessToken: tok.AccessToken,
		TokenType:   tokenType,
	}
	if tok.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	}
	return token, nil
}
