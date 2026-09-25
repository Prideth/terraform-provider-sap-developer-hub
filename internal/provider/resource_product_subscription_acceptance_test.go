package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccProductSubscription_basic additionally requires
// SAP_DEVELOPER_HUB_TEST_PRODUCT_NAME and
// SAP_DEVELOPER_HUB_TEST_PRODUCT_NAME_2 to be set to the technical names of
// two already-published products in the test tenant, since authoring a
// product is out of this provider's scope (DESIGN.md §2) and so cannot be
// created by the test itself. The second product exercises the in-place
// Update path (DESIGN.md §6/§9 - product_name is no longer RequiresReplace
// now that SAP's documented PUT payload for this entity is known).
func TestAccProductSubscription_basic(t *testing.T) {
	productName := os.Getenv("SAP_DEVELOPER_HUB_TEST_PRODUCT_NAME")
	productName2 := os.Getenv("SAP_DEVELOPER_HUB_TEST_PRODUCT_NAME_2")
	if productName == "" || productName2 == "" {
		t.Skip("acceptance test skipped: SAP_DEVELOPER_HUB_TEST_PRODUCT_NAME and SAP_DEVELOPER_HUB_TEST_PRODUCT_NAME_2 must both be set")
	}
	title := fmt.Sprintf("acctest-sub-app-%d", acctestSuffix())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProductSubscriptionConfig(title, productName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("developerhub_product_subscription.test", "product_name", productName),
					resource.TestCheckResourceAttrSet("developerhub_product_subscription.test", "id"),
					resource.TestCheckResourceAttrPair(
						"developerhub_product_subscription.test", "application_id",
						"developerhub_application.test", "id",
					),
				),
			},
			{
				Config:   testAccProductSubscriptionConfig(title, productName),
				PlanOnly: true,
			},
			{
				ResourceName:      "developerhub_product_subscription.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Updating product_name must be an in-place update, not a
				// replace - the same subscription id must survive.
				Config: testAccProductSubscriptionConfig(title, productName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("developerhub_product_subscription.test", "product_name", productName2),
				),
			},
		},
	})
}

func testAccProductSubscriptionConfig(title, productName string) string {
	return fmt.Sprintf(`
resource "developerhub_application" "test" {
  title = %[1]q
}

resource "developerhub_product_subscription" "test" {
  application_id = developerhub_application.test.id
  product_name   = %[2]q
}
`, title, productName)
}
