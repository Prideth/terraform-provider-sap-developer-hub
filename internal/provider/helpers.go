package provider

import (
	"errors"
	"fmt"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/apierror"
)

// diagnosticDetail renders err for a diag.Diagnostics AddError detail,
// unwrapping a *apierror.Error to show the SAP status code, error code and
// message (DESIGN.md §14) instead of a generic Go error string.
func diagnosticDetail(err error) string {
	var apiErr *apierror.Error
	if errors.As(err, &apiErr) {
		if apiErr.Code != "" {
			return fmt.Sprintf("SAP API returned HTTP %d, code %q:\n%s", apiErr.StatusCode, apiErr.Code, apiErr.Message)
		}
		return fmt.Sprintf("SAP API returned HTTP %d:\n%s", apiErr.StatusCode, apiErr.Message)
	}
	return err.Error()
}

// isNotFound reports whether err represents a 404 from the Developer Hub
// API, unwrapping a *apierror.Error if present.
func isNotFound(err error) bool {
	var apiErr *apierror.Error
	return errors.As(err, &apiErr) && apiErr.IsNotFound()
}
