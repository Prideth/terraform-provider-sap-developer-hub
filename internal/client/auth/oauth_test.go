package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func newTokenServer(t *testing.T, tokenCounter *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Fatalf("unexpected token request path %q", r.URL.Path)
		}
		n := tokenCounter.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "token-" + itoa(n),
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	}))
}

func itoa(n int32) string {
	digits := "0123456789"
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 4)
	for n > 0 {
		buf = append([]byte{digits[n%10]}, buf...)
		n /= 10
	}
	return string(buf)
}

func TestHTTPClient_CachesToken(t *testing.T) {
	var tokenCalls atomic.Int32
	tokenServer := newTokenServer(t, &tokenCalls)
	defer tokenServer.Close()

	var seenAuth []string
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = append(seenAuth, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer apiServer.Close()

	cfg := Config{TokenURL: tokenServer.URL + "/oauth/token", ClientID: "id", ClientSecret: "secret"}
	client, _, err := cfg.HTTPClient(context.Background(), http.DefaultClient)
	if err != nil {
		t.Fatalf("HTTPClient: %v", err)
	}

	for i := 0; i < 3; i++ {
		resp, err := client.Get(apiServer.URL)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		_ = resp.Body.Close()
	}

	if got := tokenCalls.Load(); got != 1 {
		t.Fatalf("expected exactly 1 token fetch across 3 requests, got %d", got)
	}
	if len(seenAuth) != 3 || seenAuth[0] != seenAuth[1] || seenAuth[1] != seenAuth[2] {
		t.Fatalf("expected the same bearer token reused across requests, got %v", seenAuth)
	}
}

func TestHTTPClient_InvalidateForcesRefetch(t *testing.T) {
	var tokenCalls atomic.Int32
	tokenServer := newTokenServer(t, &tokenCalls)
	defer tokenServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer apiServer.Close()

	cfg := Config{TokenURL: tokenServer.URL + "/oauth/token", ClientID: "id", ClientSecret: "secret"}
	client, invalidate, err := cfg.HTTPClient(context.Background(), http.DefaultClient)
	if err != nil {
		t.Fatalf("HTTPClient: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, apiServer.URL, nil)
	resp, err := client.Transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	_ = resp.Body.Close()
	if got := tokenCalls.Load(); got != 1 {
		t.Fatalf("expected 1 token fetch, got %d", got)
	}

	invalidate()

	resp, err = client.Transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	_ = resp.Body.Close()
	if got := tokenCalls.Load(); got != 2 {
		t.Fatalf("expected invalidate() to force a second token fetch, got %d", got)
	}
}
