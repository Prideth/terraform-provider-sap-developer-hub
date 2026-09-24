// Package provider implements the Terraform Plugin Framework glue for the
// SAP Integration Suite Developer Hub provider. It never talks HTTP
// directly - it only builds a developerhub.Client in Configure and calls
// its methods, mapping results to and from Terraform model structs. See
// DESIGN.md for the API research and scope decisions behind every resource
// and data source registered here.
package provider

import (
	"context"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/auth"
	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/developerhub"
	sapthttp "github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/http"
)

// Data is what Configure stores in ProviderData, and what every
// resource/data source's own Configure method reads back.
type Data struct {
	Client  *developerhub.Client
	Version string
}

type developerHubProvider struct {
	version string
}

// New returns the provider factory main.go passes to providerserver.Serve.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &developerHubProvider{version: version}
	}
}

type providerModel struct {
	URL          types.String `tfsdk:"url"`
	TokenURL     types.String `tfsdk:"token_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

func (p *developerHubProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "developerhub"
	resp.Version = p.version
}

func (p *developerHubProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages SAP Integration Suite Developer Hub applications, custom attributes and " +
			"product subscriptions. It does not provision SAP BTP or Integration Suite itself, and does " +
			"not author products/APIs - see the provider's DESIGN.md for the scope boundary with " +
			"terraform-provider-integration-suite.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Optional: true,
				Description: "The Developer Hub application URL from a \"devportal-apiaccess\" service key's " +
					"\"url\" field. Falls back to the SAP_DEVELOPER_HUB_URL environment variable.",
			},
			"token_url": schema.StringAttribute{
				Optional: true,
				Description: "The OAuth2 token endpoint from the service key's \"tokenUrl\" field. Falls back " +
					"to the SAP_DEVELOPER_HUB_TOKEN_URL environment variable.",
			},
			"client_id": schema.StringAttribute{
				Optional: true,
				Description: "The OAuth2 client id from the service key's \"clientId\" field. Falls back to " +
					"the SAP_DEVELOPER_HUB_CLIENT_ID environment variable.",
			},
			"client_secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "The OAuth2 client secret from the service key's \"clientSecret\" field. Falls " +
					"back to the SAP_DEVELOPER_HUB_CLIENT_SECRET environment variable.",
			},
		},
	}
}

func (p *developerHubProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := stringOrEnv(config.URL, "SAP_DEVELOPER_HUB_URL")
	tokenURL := stringOrEnv(config.TokenURL, "SAP_DEVELOPER_HUB_TOKEN_URL")
	clientID := stringOrEnv(config.ClientID, "SAP_DEVELOPER_HUB_CLIENT_ID")
	clientSecret := stringOrEnv(config.ClientSecret, "SAP_DEVELOPER_HUB_CLIENT_SECRET")

	if url == "" || tokenURL == "" || clientID == "" || clientSecret == "" {
		resp.Diagnostics.AddError(
			"Incomplete Developer Hub provider configuration",
			"The \"url\", \"token_url\", \"client_id\" and \"client_secret\" attributes (or their "+
				"SAP_DEVELOPER_HUB_* environment variable equivalents) must all be set.\n\n"+
				"If you have not created a \"devportal-apiaccess\" service instance and key yet, see "+
				"DESIGN.md and the provider documentation for how to do so. If Developer Hub is not yet "+
				"reachable for this tenant at all, enable it through the SAP Integration Suite provider "+
				"first: Developer Hub is not available for the configured tenant until the Developer Hub "+
				"capability has been activated.",
		)
		return
	}

	oauthConfig := auth.Config{TokenURL: tokenURL, ClientID: clientID, ClientSecret: clientSecret}
	authenticatedClient, invalidateToken, err := oauthConfig.HTTPClient(ctx, http.DefaultClient)
	if err != nil {
		resp.Diagnostics.AddError("Unable to configure Developer Hub authentication", err.Error())
		return
	}

	httpClient := sapthttp.New(sapthttp.Config{
		Transport:       authenticatedClient,
		UserAgent:       sapthttp.UserAgent(p.version),
		InvalidateToken: invalidateToken,
	})

	data := &Data{
		Client:  developerhub.New(httpClient, url),
		Version: p.version,
	}

	resp.DataSourceData = data
	resp.ResourceData = data
}

func stringOrEnv(value types.String, envVar string) string {
	if !value.IsNull() && !value.IsUnknown() && value.ValueString() != "" {
		return value.ValueString()
	}
	return os.Getenv(envVar)
}

func (p *developerHubProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewApplicationResource,
		NewProductSubscriptionResource,
	}
}

func (p *developerHubProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCurrentUserDataSource,
		NewRegisteredUsersDataSource,
	}
}

// requireClient is called by every resource/data source's own Configure. It
// exists so the "Developer Hub is not reachable" message (DESIGN.md §2) is
// worded consistently no matter which resource or data source triggers it.
func requireClient(data *Data, kind string, diags *diag.Diagnostics) (*developerhub.Client, bool) {
	if data == nil || data.Client == nil {
		diags.AddError(
			"Developer Hub client not configured",
			"Expected a configured Developer Hub API client on the provider "+kind+", but none was set. "+
				"This usually means the provider's \"url\", \"token_url\", \"client_id\" and \"client_secret\" "+
				"were not all provided - see the provider configuration error above, or DESIGN.md.",
		)
		return nil, false
	}
	return data.Client, true
}

// dataFromProviderData extracts *Data from req.ProviderData, adding a
// diagnostic if it is set to something unexpected. Returns ok=false if
// req.ProviderData was nil (Terraform has not called the provider's own
// Configure yet, e.g. during certain validate-only operations) so the
// caller can return early without an error.
func dataFromProviderData(raw any, diags *diag.Diagnostics) (*Data, bool) {
	if raw == nil {
		return nil, false
	}
	data, ok := raw.(*Data)
	if !ok {
		diags.AddError(
			"Unexpected provider data type",
			"Expected *provider.Data, got a different type. This is a bug in the provider.",
		)
		return nil, false
	}
	return data, true
}

// pathRoot is a small convenience wrapper used by ImportState implementations.
func pathRoot(name string) path.Path {
	return path.Root(name)
}
