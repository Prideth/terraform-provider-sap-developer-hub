package developerhub

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/auth"
	sapthttp "github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/http"
)

// TestAccServiceContract validates Contract against the live $metadata of the
// Developer Hub tenant configured through SAP_DEVELOPER_HUB_*. It is the
// provider's check against the newest API version actually deployed
// (DESIGN.md §20): if SAP renames or removes anything the provider uses, this
// fails with the exact entity set and field. Requires TF_ACC=1 and real
// credentials; skipped otherwise.
func TestAccServiceContract(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test skipped: TF_ACC is not set")
	}
	env := map[string]string{}
	for _, name := range []string{
		"SAP_DEVELOPER_HUB_URL",
		"SAP_DEVELOPER_HUB_TOKEN_URL",
		"SAP_DEVELOPER_HUB_CLIENT_ID",
		"SAP_DEVELOPER_HUB_CLIENT_SECRET",
	} {
		v := os.Getenv(name)
		if v == "" {
			t.Skipf("acceptance test skipped: %s is not set", name)
		}
		env[name] = v
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	authClient, invalidate, err := auth.Config{
		TokenURL:     env["SAP_DEVELOPER_HUB_TOKEN_URL"],
		ClientID:     env["SAP_DEVELOPER_HUB_CLIENT_ID"],
		ClientSecret: env["SAP_DEVELOPER_HUB_CLIENT_SECRET"],
	}.HTTPClient(ctx, http.DefaultClient)
	if err != nil {
		t.Fatalf("building authenticated client: %v", err)
	}
	client := New(sapthttp.New(sapthttp.Config{
		Transport:       authClient,
		UserAgent:       sapthttp.UserAgent("acctest"),
		InvalidateToken: invalidate,
	}), env["SAP_DEVELOPER_HUB_URL"])

	meta, err := client.GetServiceMetadata(ctx)
	if err != nil {
		t.Fatalf("fetching live $metadata: %v", err)
	}
	if problems := ValidateContract(meta); len(problems) != 0 {
		t.Fatalf("the live Developer Hub API no longer matches what this provider uses:\n  %s",
			strings.Join(problems, "\n  "))
	}
}
