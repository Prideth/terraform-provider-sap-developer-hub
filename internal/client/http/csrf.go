package http

import (
	"context"
	"net/http"
	"sync"
)

// SAP's Gateway-style OData services (which the Developer Hub API's
// /odata/1.0/data.svc/ endpoint is one instance of, per the sample write
// payloads in DESIGN.md §5/§15) CSRF-protect POST/PUT/PATCH/DELETE
// independently of OAuth: a client must first GET any URL on the service
// with "X-CSRF-Token: fetch", then send the returned token (and any
// Set-Cookie values) back on the write request.
type csrfCache struct {
	mu     sync.RWMutex
	token  string
	cookie string
}

func (c *csrfCache) apply(req *http.Request) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.token == "" {
		return
	}
	req.Header.Set("X-CSRF-Token", c.token)
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
}

func (c *csrfCache) store(token, cookie string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
	c.cookie = cookie
}

func requiresCSRFToken(resp *http.Response) bool {
	return resp.Header.Get("X-CSRF-Token") == "Required"
}

// refreshCSRFToken issues a GET against the same URL as req with
// "X-CSRF-Token: fetch" and caches the token and cookies returned.
func (c *Client) refreshCSRFToken(ctx context.Context, req *http.Request) error {
	fetchReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL.String(), nil)
	if err != nil {
		return err
	}
	fetchReq.Header.Set("X-CSRF-Token", "fetch")
	fetchReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.doer.Do(fetchReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	token := resp.Header.Get("X-CSRF-Token")
	cookie := resp.Header.Get("Set-Cookie")
	if token != "" {
		c.csrf.store(token, cookie)
	}
	return nil
}
