package developerhub

import "encoding/json"

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
}

func (c *odataCollection[T]) UnmarshalJSON(data []byte) error {
	var asArray []T
	if err := json.Unmarshal(data, &asArray); err == nil {
		c.Items = asArray
		return nil
	}

	var asResults struct {
		Results []T `json:"results"`
	}
	if err := json.Unmarshal(data, &asResults); err != nil {
		return err
	}
	c.Items = asResults.Results
	return nil
}
