// Package developerhub is the domain client for the SAP Integration Suite
// Developer Hub programmatic API (the "devportal-apiaccess" service plan
// described in DESIGN.md §4). It never talks Terraform; it only knows how
// to call the Developer Hub REST/OData endpoints and map results to and
// from Go structs.
package developerhub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	sapthttp "github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/http"
)

// Client is a thin JSON REST client over the Developer Hub API host.
type Client struct {
	http    *sapthttp.Client
	baseURL string
}

// New builds a Client. baseURL is the "url" value from a devportal-apiaccess
// service key, with no trailing slash.
func New(httpClient *sapthttp.Client, baseURL string) *Client {
	return &Client{
		http:    httpClient,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (c *Client) url(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.baseURL + path
}

// send issues a JSON request and decodes a JSON response into out (which
// may be nil for calls that return no body, such as DELETE). A non-2xx
// response is converted into *apierror.Error via parseError.
func (c *Client) send(ctx context.Context, method, path string, body any, out any) error {
	var payload []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding Developer Hub API request body: %w", err)
		}
		payload = encoded
	}

	req, err := sapthttp.NewRequest(ctx, method, c.url(path), payload)
	if err != nil {
		return fmt.Errorf("building Developer Hub API request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("calling Developer Hub API: %w", err)
	}

	respBody, err := sapthttp.ReadLimited(resp)
	if err != nil {
		return fmt.Errorf("reading Developer Hub API response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return parseError(resp.StatusCode, respBody)
	}

	if out == nil || len(bytes.TrimSpace(respBody)) == 0 {
		return nil
	}

	if err := decodeEnvelope(respBody, out); err != nil {
		return fmt.Errorf("decoding Developer Hub API response: %w", err)
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.send(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	return c.send(ctx, http.MethodPost, path, body, out)
}

func (c *Client) put(ctx context.Context, path string, body any) error {
	return c.send(ctx, http.MethodPut, path, body, nil)
}

func (c *Client) delete(ctx context.Context, path string) error {
	return c.send(ctx, http.MethodDelete, path, nil, nil)
}

// odataKey renders a single-quoted OData string key, escaping embedded
// single quotes by doubling them (the standard OData v1/v2 escaping rule).
func odataKey(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
