package developerhub

import (
	"context"
	"fmt"
)

const applicationsPath = "/odata/1.0/data.svc/APIMgmt.Applications"

// Application is a Developer Hub application, as modeled by the
// "APIMgmt.Applications" OData entity documented in DESIGN.md §5
// (custom-attributes-90a5a6d.md, add-custom-attributes-to-an-application-39c3cbd.md).
// Field names and casing (id, version, title, developer_id) are taken
// verbatim from SAP's sample create payload.
type Application struct {
	ID          string `json:"id,omitempty"`
	Version     string `json:"version,omitempty"`
	Title       string `json:"title"`
	DeveloperID string `json:"developer_id,omitempty"`
}

// CreateApplication creates a new Developer Hub application and returns it
// with the server-assigned id populated.
func (c *Client) CreateApplication(ctx context.Context, app Application) (*Application, error) {
	app.ID = ""
	var created Application
	if err := c.post(ctx, applicationsPath, app, &created); err != nil {
		return nil, err
	}
	if created.ID == "" {
		// Defensive fallback: some SAP OData services only return the
		// created entity's key in a Location header rather than the body.
		// Since that was not observed in the available documentation, this
		// branch just surfaces what the API gave back rather than guessing
		// further, and Read (called right after Create by the resource)
		// will fail loudly if the id genuinely never came back.
		return &created, nil
	}
	return &created, nil
}

// GetApplication reads a single application by its SAP-assigned id. Its
// custom attributes are read separately via ListApplicationAttributes
// (attributes.go) and its subscriptions via ListApplicationSubscriptions
// (subscriptions.go), since those are documented as their own navigation
// collections rather than fields guaranteed to be inlined on the
// application entity itself.
func (c *Client) GetApplication(ctx context.Context, id string) (*Application, error) {
	path := fmt.Sprintf("%s(%s)", applicationsPath, odataKey(id))
	var app Application
	if err := c.get(ctx, path, &app); err != nil {
		return nil, err
	}
	return &app, nil
}

// UpdateApplication updates the mutable fields of an existing application.
func (c *Client) UpdateApplication(ctx context.Context, id string, app Application) error {
	path := fmt.Sprintf("%s(%s)", applicationsPath, odataKey(id))
	return c.put(ctx, path, app)
}

// DeleteApplication deletes an application by id.
func (c *Client) DeleteApplication(ctx context.Context, id string) error {
	path := fmt.Sprintf("%s(%s)", applicationsPath, odataKey(id))
	return c.delete(ctx, path)
}
