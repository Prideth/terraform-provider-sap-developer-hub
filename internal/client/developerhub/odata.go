package developerhub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// decodeEnvelope decodes a Developer Hub API JSON response into out.
//
// SAP's classic OData v1/v2 Gateway services wrap a response payload in a
// top-level "d" property (a convention meant to defend against a
// JavaScript array-constructor vulnerability in old browsers). Whether the
// Developer Hub "/odata/1.0/data.svc/" endpoint does the same was not
// independently confirmed during research (DESIGN.md §5/§13 — the SAP
// documentation available showed only the request payload shapes, not a
// full response example). decodeEnvelope therefore unwraps a "d" property
// when present and falls back to decoding the body directly otherwise, so
// either shape works without the caller needing to know which one SAP
// actually returns.
func decodeEnvelope(body []byte, out any) error {
	var wrapper struct {
		D json.RawMessage `json:"d"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil && len(wrapper.D) > 0 {
		return json.Unmarshal(wrapper.D, out)
	}
	return json.Unmarshal(body, out)
}

// odataCollection decodes either a bare JSON array or an OData v2-style
// {"results": [...]} object into Items, for the same reason decodeEnvelope
// tolerates both shapes of the outer envelope.
type odataCollection[T any] struct {
	Items []T
	// Next is the OData v2 server-driven paging link ("__next"), empty on
	// the last page or when the service does not page.
	Next string
}

func (c *odataCollection[T]) UnmarshalJSON(data []byte) error {
	var asArray []T
	if err := json.Unmarshal(data, &asArray); err == nil {
		c.Items = asArray
		c.Next = ""
		return nil
	}

	var asResults struct {
		Results []T    `json:"results"`
		Next    string `json:"__next"`
	}
	if err := json.Unmarshal(data, &asResults); err != nil {
		return err
	}
	c.Items = asResults.Results
	c.Next = asResults.Next
	return nil
}

// maxPages bounds listAll so a service that keeps returning a __next link
// cannot keep the provider looping forever.
const maxPages = 1000

// listAll reads every page of an OData collection, following "__next"
// links. A next link is only followed if it points at the configured
// Developer Hub host: every request carries the bearer token, so a link to
// any other host is refused rather than followed.
func listAll[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	var all []T
	for page := 0; page < maxPages; page++ {
		var collection odataCollection[T]
		if err := c.get(ctx, path, &collection); err != nil {
			return nil, err
		}
		all = append(all, collection.Items...)
		if collection.Next == "" {
			return all, nil
		}
		next, err := c.resolveNextLink(path, collection.Next)
		if err != nil {
			return nil, err
		}
		path = next
	}
	return nil, fmt.Errorf("developer hub API returned more than %d pages for %s", maxPages, path)
}

// resolveNextLink turns a "__next" link into a path on the configured host.
// OData services return it either absolute or relative to the collection's
// URL; both are resolved against the current request URL.
func (c *Client) resolveNextLink(currentPath, next string) (string, error) {
	base, err := url.Parse(c.url(currentPath))
	if err != nil {
		return "", fmt.Errorf("parsing request URL: %w", err)
	}
	ref, err := url.Parse(next)
	if err != nil {
		return "", fmt.Errorf("parsing paging link %q: %w", next, err)
	}
	resolved := base.ResolveReference(ref)
	configured, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("parsing configured URL: %w", err)
	}
	if resolved.Scheme != configured.Scheme || resolved.Host != configured.Host {
		return "", fmt.Errorf("refusing to follow paging link to a different host (%s)", resolved.Host)
	}
	return resolved.String(), nil
}
