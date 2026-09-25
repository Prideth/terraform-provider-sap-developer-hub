// Package http provides the shared retrying HTTP transport used by every
// Developer Hub domain client: exponential backoff with jitter on
// transient failures, Retry-After handling, a single token-refresh retry on
// 401, CSRF token handling for the OData write operations the Developer
// Hub API requires (see DESIGN.md §15), a bounded response size, and a
// stable User-Agent.
package http

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultMaxRetries = 4
	defaultBaseDelay  = 250 * time.Millisecond
	defaultMaxDelay   = 30 * time.Second

	// MaxResponseBytes bounds how much of a response body is ever read into
	// memory or surfaced in an error message.
	MaxResponseBytes = 64 << 20 // 64 MiB
)

// Doer is the minimal interface Client needs from an *http.Client, so tests
// can substitute a fake.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Config configures a Client.
type Config struct {
	// Transport is the authenticated HTTP client (see internal/client/auth)
	// requests are sent through.
	Transport Doer
	// UserAgent is sent on every request.
	UserAgent string
	// InvalidateToken is called once after a 401, before the single retry
	// that follows it. May be nil.
	InvalidateToken func()
	// MaxRetries overrides defaultMaxRetries. Zero means "use the default".
	MaxRetries int
}

// Client wraps a Doer with retry, backoff, and CSRF handling.
type Client struct {
	doer            Doer
	userAgent       string
	invalidateToken func()
	maxRetries      int
	csrf            csrfCache
}

// UserAgent builds the provider's standard User-Agent string.
func UserAgent(providerVersion string) string {
	return fmt.Sprintf("terraform-provider-sap-developer-hub/%s", providerVersion)
}

// New builds a Client from Config.
func New(cfg Config) *Client {
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}
	return &Client{
		doer:            cfg.Transport,
		userAgent:       cfg.UserAgent,
		invalidateToken: cfg.InvalidateToken,
		maxRetries:      maxRetries,
	}
}

// Do sends req, retrying on transient failures and handling CSRF tokens for
// write methods. req.GetBody must be set (or req.Body nil) for a request
// that might need to be retried; NewRequest below takes care of that for
// callers that build requests through it.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", c.userAgent)

	if isWriteMethod(req.Method) {
		c.csrf.apply(req)
	}

	var lastErr error
	invalidatedToken := false

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := c.wait(ctx, attempt, lastErr, nil); err != nil {
				return nil, err
			}
			if err := rewindBody(req); err != nil {
				return nil, err
			}
		}

		resp, err := c.doer.Do(req)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			continue
		}

		if isWriteMethod(req.Method) && resp.StatusCode == http.StatusForbidden && requiresCSRFToken(resp) {
			_ = resp.Body.Close()
			if err := c.refreshCSRFToken(ctx, req); err != nil {
				return nil, err
			}
			c.csrf.apply(req)
			if err := rewindBody(req); err != nil {
				return nil, err
			}
			resp, err = c.doer.Do(req)
			if err != nil {
				lastErr = err
				continue
			}
		}

		if resp.StatusCode == http.StatusUnauthorized && !invalidatedToken && c.invalidateToken != nil {
			_ = resp.Body.Close()
			c.invalidateToken()
			invalidatedToken = true
			if err := rewindBody(req); err != nil {
				return nil, err
			}
			resp, err = c.doer.Do(req)
			if err != nil {
				lastErr = err
				continue
			}
		}

		if isRetryable(resp.StatusCode) && attempt < c.maxRetries {
			retryAfter := retryAfterDelay(resp)
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("transient HTTP %d", resp.StatusCode)
			if retryAfter > 0 {
				if err := c.wait(ctx, attempt+1, nil, &retryAfter); err != nil {
					return nil, err
				}
				if err := rewindBody(req); err != nil {
					return nil, err
				}
				resp, err = c.doer.Do(req)
				if err != nil {
					lastErr = err
					continue
				}
				return resp, nil
			}
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

func isWriteMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func isRetryable(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func retryAfterDelay(resp *http.Response) time.Duration {
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0
	}
	seconds, err := strconv.Atoi(v)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// wait sleeps for either the explicit override (used for Retry-After), or a
// full-jitter exponential backoff computed from the attempt number.
func (c *Client) wait(ctx context.Context, attempt int, _ error, override *time.Duration) error {
	delay := override
	if delay == nil {
		d := backoffDelay(attempt)
		delay = &d
	}

	timer := time.NewTimer(*delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func backoffDelay(attempt int) time.Duration {
	max := defaultBaseDelay << attempt
	if max > defaultMaxDelay || max <= 0 {
		max = defaultMaxDelay
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return max / 2
	}
	return time.Duration(n.Int64())
}

func rewindBody(req *http.Request) error {
	if req.GetBody == nil {
		return nil
	}
	body, err := req.GetBody()
	if err != nil {
		return fmt.Errorf("rewinding request body for retry: %w", err)
	}
	req.Body = body
	return nil
}

// NewRequest builds an *http.Request with GetBody populated so the request
// can be safely retried.
func NewRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))
	return req, nil
}

// ReadLimited reads resp.Body up to MaxResponseBytes, closing it, and
// returns the bytes read.
func ReadLimited(resp *http.Response) ([]byte, error) {
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes))
}
