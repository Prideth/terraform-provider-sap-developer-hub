package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDeveloper_basic registers a real identity-provider user as a
// Developer Hub developer and revokes the registration again on destroy. It
// needs SAP_DEVELOPER_HUB_TEST_DEVELOPER_USER_ID and
// SAP_DEVELOPER_HUB_TEST_DEVELOPER_EMAIL for a user that exists in the test
// subaccount's identity provider and is not registered yet.
func TestAccDeveloper_basic(t *testing.T) {
	userID := os.Getenv("SAP_DEVELOPER_HUB_TEST_DEVELOPER_USER_ID")
	email := os.Getenv("SAP_DEVELOPER_HUB_TEST_DEVELOPER_EMAIL")
	if userID == "" || email == "" {
		t.Skip("acceptance test skipped: SAP_DEVELOPER_HUB_TEST_DEVELOPER_USER_ID and SAP_DEVELOPER_HUB_TEST_DEVELOPER_EMAIL must both be set")
	}

	config := fmt.Sprintf(`
resource "developerhub_developer" "test" {
  user_id           = %[1]q
  email             = %[2]q
  first_name        = "Acceptance"
  last_name         = "Test"
  revocation_reason = "Removed by an acceptance test"
}
`, userID, email)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("developerhub_developer.test", "id", userID),
					resource.TestCheckResourceAttr("developerhub_developer.test", "status", "registered"),
				),
			},
			{Config: config, PlanOnly: true},
			{
				ResourceName:            "developerhub_developer.test",
				ImportState:             true,
				ImportStateId:           userID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"revocation_reason"},
			},
		},
	})
}
