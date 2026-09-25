package developerhub

import "context"

const productsPath = "/odata/1.0/data.svc/APIMgmt.APIProducts"

// Product is a product published to the Developer Hub catalog:
// "developer.APIProductsType" in the DevPortal_Application_CF specification
// (api-specs/), keyed by name. Products are authored in SAP Integration
// Suite, not by this provider (DESIGN.md §2); this is read-only.
type Product struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Version     string `json:"version"`
	Vendor      string `json:"vendor,omitempty"`
	Description string `json:"description,omitempty"`
	ShortText   string `json:"shortText,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	PublishedBy string `json:"published_by,omitempty"`
}

// ListProducts reads every product in the APIProducts entity set, which the
// specification declares in the service's APIMgmt entity container.
func (c *Client) ListProducts(ctx context.Context) ([]Product, error) {
	return listAll[Product](ctx, c, productsPath)
}
