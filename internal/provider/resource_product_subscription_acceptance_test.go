package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccProductSubscription_basic additionally requires
// SAP_DEVELOPER_HUB_TEST_PRODUCT_NAME to be set to the technical name of an
// already-published product in the test tenant, since authoring a product
// is out of this provider's scope (DESIGN.md §2) and so cannot be created
// by the test itself.
func TestAccProductSubscription_basic(t *testing.T) {
	productName := os.Getenv("SAP_DEVELOPER_HUB_TEST_PRODUCT_NAME")
	if productName == "" {
		t.Skip("acceptance test skipped: SAP_DEVELOPER_HUB_TEST_PRODUCT_NAME is not set")
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
				ImportStateIdFunc: testAccProductSubscriptionImportStateIDFunc("developerhub_product_subscription.test"),
			},
		},
	})
}

// testAccProductSubscriptionImportStateIDFunc builds the
// "<application_id>/<subscription_id>" import identifier DESIGN.md §8
// documents, from the resource's own state, since it can't be known ahead
// of the test run.
func testAccProductSubscriptionImportStateIDFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found in state: %s", resourceName)
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["application_id"], rs.Primary.ID), nil
	}
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
