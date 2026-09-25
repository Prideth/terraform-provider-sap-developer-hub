package developerhub

import "context"

const subscriptionsPath = "/odata/1.0/data.svc/APIMgmt.Subscriptions"

// Subscription is a Developer Hub product subscription, modeled after the
// "APIMgmt.Subscriptions" OData entity documented in
// create-or-update-or-read-an-application-using-subscription-key-e2645b5.md
// (DESIGN.md §5/§9). It is its own top-level, independently addressable
// entity set (key: id) - not a navigation collection nested under
// Applications, which an earlier reading of a different SAP document had
// suggested (see DESIGN.md §9 for the correction and its evidence).
//
// Field names (id, app_id, product_id, developer_id, isSubscribed, status)
// are taken verbatim from SAP's own EDMX for this entity type and from its
// sample update payload. AppID and ProductID are populated directly from
// the entity's own plain fields on Read/Update; on Create, SAP's documented
// payload sets the relationship via the ToApplication/ToAPIProduct
// navigation properties instead (see CreateSubscription).
type Subscription struct {
	ID           string `json:"id,omitempty"`
	AppID        string `json:"app_id,omitempty"`
	ProductID    string `json:"product_id,omitempty"`
	DeveloperID  string `json:"developer_id,omitempty"`
	IsSubscribed bool   `json:"isSubscribed"`
	// Status reflects SAP's own subscription governance lifecycle (e.g. a
	// pending-approval state under External Governance - see DESIGN.md
	// §9/setting-up-external-governance-992512e.md). Its exact set of wire
	// values was not confirmed, so this provider treats it as an opaque,
	// read-only string rather than validating or normalizing it.
	Status string `json:"status,omitempty"`
}

type subscriptionWriteRequest struct {
	ID          string             `json:"id,omitempty"`
	ProductID   string             `json:"product_id,omitempty"`
	APIProduct  []apiProductRefIn  `json:"ToAPIProduct,omitempty"`
	Application []applicationRefIn `json:"ToApplication,omitempty"`
}

type apiProductRefIn struct {
	Metadata odataMetadata `json:"__metadata"`
}

type applicationRefIn struct {
	Metadata odataMetadata `json:"__metadata"`
}

type odataMetadata struct {
	URI string `json:"uri"`
}

func apiProductReferenceURI(productName string) string {
	return "APIMgmt.APIProducts(" + odataKey(productName) + ")"
}

func applicationReferenceURI(applicationID string) string {
	return "APIMgmt.Applications(" + odataKey(applicationID) + ")"
}

// CreateSubscription subscribes an application to a product by the
// product's technical name (the product's stable id in the API
// Portal/Integration Suite APIProducts service this provider does not
// manage - see DESIGN.md §2), via "POST .../APIMgmt.Subscriptions" with
// ToAPIProduct/ToApplication navigation references, exactly as documented.
func (c *Client) CreateSubscription(ctx context.Context, applicationID, productName string) (*Subscription, error) {
	req := subscriptionWriteRequest{
		ID:          placeholderID,
		APIProduct:  []apiProductRefIn{{Metadata: odataMetadata{URI: apiProductReferenceURI(productName)}}},
		Application: []applicationRefIn{{Metadata: odataMetadata{URI: applicationReferenceURI(applicationID)}}},
	}
	var created Subscription
	if err := c.post(ctx, subscriptionsPath, req, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// GetSubscription reads a single subscription by its own id.
func (c *Client) GetSubscription(ctx context.Context, id string) (*Subscription, error) {
	path := subscriptionsPath + "(" + odataKey(id) + ")"
	var sub Subscription
	if err := c.get(ctx, path, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

// UpdateSubscriptionProduct changes which product an existing subscription
// targets, via "PUT .../APIMgmt.Subscriptions(<id>)" sending both the
// product_id field and the ToAPIProduct navigation reference, matching the
// documented update payload exactly.
func (c *Client) UpdateSubscriptionProduct(ctx context.Context, id, productName string) error {
	path := subscriptionsPath + "(" + odataKey(id) + ")"
	req := subscriptionWriteRequest{
		ID:         id,
		ProductID:  productName,
		APIProduct: []apiProductRefIn{{Metadata: odataMetadata{URI: apiProductReferenceURI(productName)}}},
	}
	return c.put(ctx, path, req)
}

// DeleteSubscription deletes a subscription by id. SAP's documentation
// shows DELETE for the sibling APIMgmt.Applications entity in this exact
// form; DELETE on APIMgmt.Subscriptions itself was not separately shown,
// but is the same OData service and the same addressing pattern already
// confirmed for GET/POST/PUT above - documented here as a high-confidence
// inference (DESIGN.md §9), not an invented endpoint.
func (c *Client) DeleteSubscription(ctx context.Context, id string) error {
	path := subscriptionsPath + "(" + odataKey(id) + ")"
	return c.delete(ctx, path)
}
