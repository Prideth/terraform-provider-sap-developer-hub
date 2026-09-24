package developerhub

import (
	"encoding/json"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/apierror"
)

// odataErrorEnvelope is the standard SAP Gateway OData error shape:
//
//	{"error":{"code":"...","message":{"lang":"en","value":"..."}}}
//
// This is the conventional error envelope across SAP's OData-based Cloud
// Foundry services; DESIGN.md §5/§15 documents that this was not
// independently confirmed for the Developer Hub API specifically (only the
// success payloads were shown in the SAP documentation available during
// research). parseError therefore treats it as the first, most likely
// shape to try, and falls back gracefully rather than failing to parse
// entirely, so a genuinely different error shape still surfaces the raw
// message instead of crashing.
type odataErrorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message struct {
			Value string `json:"value"`
		} `json:"message"`
	} `json:"error"`
}

// flatErrorEnvelope covers plain-JSON APIs (such as the /api/1.0/ family)
// that report an error as a flat object.
type flatErrorEnvelope struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

func parseError(statusCode int, body []byte) *apierror.Error {
	if len(body) > 0 {
		var odataErr odataErrorEnvelope
		if err := json.Unmarshal(body, &odataErr); err == nil && odataErr.Error.Message.Value != "" {
			return &apierror.Error{
				StatusCode: statusCode,
				Code:       odataErr.Error.Code,
				Message:    odataErr.Error.Message.Value,
			}
		}

		var flatErr flatErrorEnvelope
		if err := json.Unmarshal(body, &flatErr); err == nil {
			if flatErr.Message != "" {
				return &apierror.Error{StatusCode: statusCode, Message: flatErr.Message}
			}
			if flatErr.Error != "" {
				return &apierror.Error{StatusCode: statusCode, Message: flatErr.Error}
			}
		}
	}

	message := "no error details returned"
	if len(body) > 0 {
		message = truncate(string(body), 500)
	}
	return &apierror.Error{StatusCode: statusCode, Message: message}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "... (truncated)"
}
