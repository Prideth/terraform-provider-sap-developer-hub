package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The /api/1.0/ endpoints behind these data sources are plain JSON, not
// OData, so they are not covered by the $metadata contract check
// (internal/client/developerhub/contract_acceptance_test.go). These tests are
// their live validation instead: they fail if the response shape changes so
// that the fields the data sources read come back empty.

func TestAccCurrentUserDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "developerhub_current_user" "me" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.developerhub_current_user.me", "name"),
					resource.TestCheckResourceAttrSet("data.developerhub_current_user.me", "id"),
				),
			},
		},
	})
}

func TestAccRegisteredUsersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "developerhub_registered_users" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.developerhub_registered_users.all", "id", "registered_users"),
					resource.TestCheckResourceAttrSet("data.developerhub_registered_users.all", "user.#"),
				),
			},
		},
	})
}

func TestAccApplicationsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "developerhub_applications" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.developerhub_applications.all", "id", "applications"),
					resource.TestCheckResourceAttrSet("data.developerhub_applications.all", "application.#"),
				),
			},
		},
	})
}

func TestAccProductSubscriptionsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "developerhub_product_subscriptions" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.developerhub_product_subscriptions.all", "id", "product_subscriptions"),
					resource.TestCheckResourceAttrSet("data.developerhub_product_subscriptions.all", "subscription.#"),
				),
			},
		},
	})
}

func TestAccProductsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "developerhub_products" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.developerhub_products.all", "id", "products"),
					resource.TestCheckResourceAttrSet("data.developerhub_products.all", "product.#"),
				),
			},
		},
	})
}

func TestAccRegistrationRequestsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "developerhub_registration_requests" "pending" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.developerhub_registration_requests.pending", "id", "registration_requests"),
					resource.TestCheckResourceAttrSet("data.developerhub_registration_requests.pending", "request.#"),
				),
			},
		},
	})
}
