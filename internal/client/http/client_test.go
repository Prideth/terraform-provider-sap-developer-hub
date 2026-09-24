package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(doer Doer) *Client {
	return New(Config{Transport: doer, UserAgent: "test-agent", MaxRetries: 3})
}

func TestDo_RetriesOnServiceUnavailable(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := testClient(http.DefaultClient)
	req, _ := NewRequest(context.Background(), http.MethodGet, server.URL, nil)

	start := time.Now()
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected eventual 200, got %d", resp.StatusCode)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", calls.Load())
	}
	if time.Since(start) > 5*time.Second {
		t.Fatalf("retries took too long: %v", time.Since(start))
	}
}

func TestDo_HonorsRetryAfter(t *testing.T) {
	var calls atomic.Int32
	var firstCallTime time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			firstCallTime = time.Now()
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if time.Since(firstCallTime) < 900*time.Millisecond {
			t.Errorf("retry happened too soon: %v after first call", time.Since(firstCallTime))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := testClient(http.DefaultClient)
	req, _ := NewRequest(context.Background(), http.MethodGet, server.URL, nil)

	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected eventual 200, got %d", resp.StatusCode)
	}
}

func TestDo_DoesNotRetryOnNotFound(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := testClient(http.DefaultClient)
	req, _ := NewRequest(context.Background(), http.MethodGet, server.URL, nil)

	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected exactly 1 attempt for a non-retryable status, got %d", calls.Load())
	}
}

func TestDo_InvalidatesTokenOnceOn401(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var invalidated atomic.Int32
	c := New(Config{
		Transport:       http.DefaultClient,
		UserAgent:       "test-agent",
		InvalidateToken: func() { invalidated.Add(1) },
	})
	req, _ := NewRequest(context.Background(), http.MethodGet, server.URL, nil)

	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after token refresh, got %d", resp.StatusCode)
	}
	if invalidated.Load() != 1 {
		t.Fatalf("expected InvalidateToken to be called exactly once, got %d", invalidated.Load())
	}
	if calls.Load() != 2 {
		t.Fatalf("expected exactly 2 attempts (original + 1 retry), got %d", calls.Load())
	}
}

func TestDo_FetchesAndRetriesOnCSRFRequired(t *testing.T) {
	var writeCalls atomic.Int32
	var fetchCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-CSRF-Token") == "fetch" {
			fetchCalls.Add(1)
			w.Header().Set("X-CSRF-Token", "csrf-token-value")
			w.Header().Set("Set-Cookie", "session=abc")
			w.WriteHeader(http.StatusOK)
			return
		}

		n := writeCalls.Add(1)
		if n == 1 {
			if r.Header.Get("X-CSRF-Token") != "" {
				t.Errorf("expected no CSRF token on first write attempt, got %q", r.Header.Get("X-CSRF-Token"))
			}
			w.Header().Set("X-CSRF-Token", "Required")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if r.Header.Get("X-CSRF-Token") != "csrf-token-value" {
			t.Errorf("expected cached CSRF token on retry, got %q", r.Header.Get("X-CSRF-Token"))
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	c := testClient(http.DefaultClient)
	req, _ := NewRequest(context.Background(), http.MethodPost, server.URL, []byte(`{}`))

	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 after CSRF retry, got %d", resp.StatusCode)
	}
	if fetchCalls.Load() != 1 {
		t.Fatalf("expected exactly 1 CSRF token fetch, got %d", fetchCalls.Load())
	}
	if writeCalls.Load() != 2 {
		t.Fatalf("expected exactly 2 write attempts, got %d", writeCalls.Load())
	}
}
