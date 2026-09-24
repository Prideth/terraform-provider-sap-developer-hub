package developerhub

import (
	"context"
	"fmt"
	"strings"
)

const subscriptionsNavigation = "ToSubscriptions"

// Subscription is a Developer Hub product subscription of an application,
// modeled after the nested "ToSubscriptions"/"ToAPIProduct" shape shown in
// the APIMgmt.Applications create sample payload
// (custom-attributes-90a5a6d.md, DESIGN.md §5/§9). Addressing a single
// subscription for Read/Delete as a keyed child of the application
// (rather than of a top-level "APIMgmt.Subscriptions" collection, which
// was never confirmed to exist) is a documented inference — see DESIGN.md
// §9.
type Subscription struct {
	ID          string             `json:"id,omitempty"`
	ProductName string             `json:"-"`
	APIProduct  []apiProductRefOut `json:"ToAPIProduct,omitempty"`
}

type apiProductRefOut struct {
	Metadata apiProductMetadata `json:"__metadata"`
}

type apiProductMetadata struct {
	URI string `json:"uri"`
}

type subscriptionCreateRequest struct {
	APIProduct []apiProductRefIn `json:"ToAPIProduct"`
}

type apiProductRefIn struct {
	Metadata apiProductMetadata `json:"__metadata"`
}

func productReferenceURI(productName string) string {
	return fmt.Sprintf("APIMgmt.APIProducts(%s)", odataKey(productName))
}

// CreateSubscription subscribes an application to a product by its
// technical name (the product's stable id in the API Portal/Integration
// Suite APIProducts service this provider does not manage — see
// DESIGN.md §2), via "POST .../APIMgmt.Applications(<id>)/ToSubscriptions".
func (c *Client) CreateSubscription(ctx context.Context, applicationID, productName string) (*Subscription, error) {
	path := fmt.Sprintf("%s(%s)/%s", applicationsPath, odataKey(applicationID), subscriptionsNavigation)
	req := subscriptionCreateRequest{
		APIProduct: []apiProductRefIn{{
			Metadata: apiProductMetadata{URI: productReferenceURI(productName)},
		}},
	}
	var created Subscription
	if err := c.post(ctx, path, req, &created); err != nil {
		return nil, err
	}
	created.ProductName = productName
	return &created, nil
}

// GetSubscription reads a single subscription of an application by its
// subscription id (see the type doc comment on the addressing inference).
func (c *Client) GetSubscription(ctx context.Context, applicationID, subscriptionID string) (*Subscription, error) {
	path := c.subscriptionKeyPath(applicationID, subscriptionID)
	var sub Subscription
	if err := c.get(ctx, path, &sub); err != nil {
		return nil, err
	}
	if len(sub.APIProduct) > 0 {
		sub.ProductName = productNameFromReferenceURI(sub.APIProduct[0].Metadata.URI)
	}
	return &sub, nil
}

// productNameFromReferenceURI extracts "Name" from a
// "APIMgmt.APIProducts('Name')" navigation reference URI, undoing the
// single-quote escaping odataKey applies.
func productNameFromReferenceURI(uri string) string {
	start := strings.Index(uri, "('")
	end := strings.LastIndex(uri, "')")
	if start < 0 || end < 0 || end <= start+2 {
		return ""
	}
	return strings.ReplaceAll(uri[start+2:end], "''", "'")
}

// DeleteSubscription deletes a subscription by id.
func (c *Client) DeleteSubscription(ctx context.Context, applicationID, subscriptionID string) error {
	return c.delete(ctx, c.subscriptionKeyPath(applicationID, subscriptionID))
}

func (c *Client) subscriptionKeyPath(applicationID, subscriptionID string) string {
	return fmt.Sprintf(
		"%s(%s)/%s(%s)",
		applicationsPath, odataKey(applicationID),
		subscriptionsNavigation, odataKey(subscriptionID),
	)
}
