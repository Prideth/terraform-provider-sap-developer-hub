package developerhub

import (
	"context"
	"fmt"
)

const attributesPath = "/odata/1.0/data.svc/APIMgmt.Attributes"

// attributeEntityType is the entityType key of every application custom
// attribute. The DevPortal_Application_CF specification
// (api-specs/DevPortal_Application_CF.yaml) states the value "is always
// 'applications'"; an older Help Portal sample capitalized it
// ("Applications"). The specification is the newer, authoritative source
// (DESIGN.md §20).
const attributeEntityType = "applications"

// Attribute is a Developer Hub custom attribute attached to an application:
// "developer.AttributeType" in DevPortal_Application_CF.yaml. The type
// declares name (max 235 characters), value (max 1024) and entityId; name
// and value are required.
type Attribute struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	EntityID string `json:"entityId,omitempty"`
}

// attributeValueUpdate is the specification's "customAttributePut" body: an
// update only ever carries the new value.
type attributeValueUpdate struct {
	Value string `json:"value"`
}

// ListApplicationAttributes lists the custom attributes of an application
// via "GET /APIMgmt.Applications('{id}')/ToAttributes".
func (c *Client) ListApplicationAttributes(ctx context.Context, applicationID string) ([]Attribute, error) {
	path := fmt.Sprintf("%s(%s)/ToAttributes", applicationsPath, odataKey(applicationID))
	return listAll[Attribute](ctx, c, path)
}

// CreateApplicationAttribute adds a custom attribute to an application via
// "POST /APIMgmt.Applications('{id}')/ToAttributes".
func (c *Client) CreateApplicationAttribute(ctx context.Context, applicationID, name, value string) error {
	path := fmt.Sprintf("%s(%s)/ToAttributes", applicationsPath, odataKey(applicationID))
	return c.post(ctx, path, Attribute{Name: name, Value: value, EntityID: applicationID}, nil)
}

// UpdateApplicationAttribute changes the value of an existing attribute via
// "PUT /APIMgmt.Attributes(name=..,entityId=..,entityType=..)".
func (c *Client) UpdateApplicationAttribute(ctx context.Context, applicationID, name, value string) error {
	return c.put(ctx, c.attributeKeyPath(applicationID, name), attributeValueUpdate{Value: value})
}

// DeleteApplicationAttribute deletes a custom attribute via
// "DELETE /APIMgmt.Attributes(name=..,entityId=..,entityType=..)".
func (c *Client) DeleteApplicationAttribute(ctx context.Context, applicationID, name string) error {
	return c.delete(ctx, c.attributeKeyPath(applicationID, name))
}

func (c *Client) attributeKeyPath(applicationID, name string) string {
	return fmt.Sprintf(
		"%s(name=%s,entityId=%s,entityType=%s)",
		attributesPath,
		odataKey(name),
		odataKey(applicationID),
		odataKey(attributeEntityType),
	)
}
