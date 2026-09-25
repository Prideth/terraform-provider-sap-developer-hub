package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccApplication_basic requires TF_ACC=1 and real
// SAP_DEVELOPER_HUB_* credentials (testAccPreCheck), and is skipped
// otherwise - see DESIGN.md and README.md "Testing".
func TestAccApplication_basic(t *testing.T) {
	title := fmt.Sprintf("acctest-app-%d", acctestSuffix())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationConfig(title, "prod"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("developerhub_application.test", "title", title),
					resource.TestCheckResourceAttr("developerhub_application.test", "attribute.0.name", "environment"),
					resource.TestCheckResourceAttr("developerhub_application.test", "attribute.0.value", "prod"),
					resource.TestCheckResourceAttrSet("developerhub_application.test", "id"),
					resource.TestCheckResourceAttrSet("developerhub_application.test", "app_key"),
					resource.TestCheckResourceAttrSet("developerhub_application.test", "app_secret"),
				),
			},
			{
				// A second apply of the same config must be a no-op plan,
				// per DESIGN.md's idempotency requirement.
				Config:   testAccApplicationConfig(title, "prod"),
				PlanOnly: true,
			},
			{
				Config: testAccApplicationConfig(title, "staging"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("developerhub_application.test", "attribute.0.value", "staging"),
				),
			},
			{
				ResourceName:      "developerhub_application.test",
				ImportState:       true,
				ImportStateVerify: true,
				// app_key/app_secret are only returned by SAP in the Create
				// response and are never fetched back by Read (DESIGN.md
				// §6/§16) - a freshly imported resource genuinely cannot
				// know them, so they are excluded from the import
				// comparison rather than expected to match.
				ImportStateVerifyIgnore: []string{"app_key", "app_secret"},
			},
		},
	})
}

func testAccApplicationConfig(title, envAttrValue string) string {
	return fmt.Sprintf(`
resource "developerhub_application" "test" {
  title        = %[1]q
  description  = "Created by an acceptance test"
  callback_url = "https://example.com/callback"

  attribute {
    name  = "environment"
    value = %[2]q
  }
}
`, title, envAttrValue)
}
