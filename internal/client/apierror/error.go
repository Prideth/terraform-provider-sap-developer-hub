// Package apierror defines the single normalized error type every Developer
// Hub API response is mapped to, regardless of which wire format the
// specific endpoint uses (plain JSON, or the OData v1.0 data service).
package apierror

import "fmt"

// Detail is one entry of a multi-error SAP response, when the API provides
// them.
type Detail struct {
	Code    string
	Message string
}

// Error is the normalized shape every failed Developer Hub API call is
// mapped to. Resource and data source code never inspects a raw HTTP
// response or a wire-format-specific error struct directly; it works with
// this type via errors.As.
type Error struct {
	StatusCode int
	Code       string
	Message    string
	Details    []Detail
	RequestID  string
}

func (e *Error) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("SAP Developer Hub API returned HTTP %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("SAP Developer Hub API returned HTTP %d, code %q: %s", e.StatusCode, e.Code, e.Message)
}

// IsNotFound reports whether the error represents a 404 response.
func (e *Error) IsNotFound() bool {
	return e != nil && e.StatusCode == 404
}

// IsConflict reports whether the error represents a 409 response.
func (e *Error) IsConflict() bool {
	return e != nil && e.StatusCode == 409
}

// IsUnauthorized reports whether the error represents a 401 response.
func (e *Error) IsUnauthorized() bool {
	return e != nil && e.StatusCode == 401
}
