package provider

import (
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// acctestSuffix returns a value unique enough to avoid colliding with
// another concurrent acceptance test run's resource names.
func acctestSuffix() int64 {
	return time.Now().UnixNano()
}

// testAccProtoV6ProviderFactories is passed to every acceptance
// resource.TestCase. It is only ever exercised when TF_ACC=1 (Terraform
// Plugin Testing's own gate) and testAccPreCheck has confirmed real
// credentials are present.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"developerhub": providerserver.NewProtocol6WithError(New("acctest")()),
}

// testAccPreCheck skips the calling acceptance test unless real Developer
// Hub credentials are available via the same SAP_DEVELOPER_HUB_*
// environment variables the provider itself reads (DESIGN.md §4). No
// credentials are ever hardcoded here or anywhere else in this repository.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, envVar := range []string{
		"SAP_DEVELOPER_HUB_URL",
		"SAP_DEVELOPER_HUB_TOKEN_URL",
		"SAP_DEVELOPER_HUB_CLIENT_ID",
		"SAP_DEVELOPER_HUB_CLIENT_SECRET",
	} {
		if os.Getenv(envVar) == "" {
			t.Skipf("acceptance test skipped: %s is not set", envVar)
		}
	}
}
