package developerhub

import (
	"context"
	"fmt"
)

const attributesPath = "/odata/1.0/data.svc/APIMgmt.Attributes"

// Attribute is a Developer Hub custom attribute attached to an application,
// modeled after the "APIMgmt.Attributes" entity sample payloads in
// custom-attributes-90a5a6d.md (DESIGN.md §5).
type Attribute struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
}

// ListApplicationAttributes lists the custom attributes of an application
// via its ToAttributes navigation property.
func (c *Client) ListApplicationAttributes(ctx context.Context, applicationID string) ([]Attribute, error) {
	path := fmt.Sprintf("%s(%s)/ToAttributes", applicationsPath, odataKey(applicationID))
	return listAll[Attribute](ctx, c, path)
}

// CreateApplicationAttribute adds a custom attribute to an application, via
// "POST .../APIMgmt.Applications(<id>)/ToAttributes" as documented.
func (c *Client) CreateApplicationAttribute(ctx context.Context, applicationID, name, value string) error {
	path := fmt.Sprintf("%s(%s)/ToAttributes", applicationsPath, odataKey(applicationID))
	attr := Attribute{
		Name:       name,
		Value:      value,
		EntityType: "Applications",
		EntityID:   applicationID,
	}
	return c.post(ctx, path, attr, nil)
}

// UpdateApplicationAttribute updates the value of an existing attribute, via
// "PUT APIMgmt.Attributes(name=..,entityId=..,entityType='Applications')" as
// documented.
func (c *Client) UpdateApplicationAttribute(ctx context.Context, applicationID, name, value string) error {
	path := c.attributeKeyPath(applicationID, name)
	attr := Attribute{
		Name:       name,
		Value:      value,
		EntityType: "Applications",
		EntityID:   applicationID,
	}
	return c.put(ctx, path, attr)
}

// DeleteApplicationAttribute deletes a custom attribute, via
// "DELETE APIMgmt.Attributes(name=..,entityId=..,entityType='Applications')"
// as documented.
func (c *Client) DeleteApplicationAttribute(ctx context.Context, applicationID, name string) error {
	return c.delete(ctx, c.attributeKeyPath(applicationID, name))
}

func (c *Client) attributeKeyPath(applicationID, name string) string {
	return fmt.Sprintf(
		"%s(name=%s,entityId=%s,entityType=%s)",
		attributesPath,
		odataKey(name),
		odataKey(applicationID),
		odataKey("Applications"),
	)
}
