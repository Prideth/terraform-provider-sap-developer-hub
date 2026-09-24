package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/developerhub"
)

var (
	_ datasource.DataSource              = &currentUserDataSource{}
	_ datasource.DataSourceWithConfigure = &currentUserDataSource{}
)

// NewCurrentUserDataSource is registered in provider.go's DataSources().
func NewCurrentUserDataSource() datasource.DataSource {
	return &currentUserDataSource{}
}

type currentUserDataSource struct {
	client *developerhub.Client
}

type currentUserModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	Email     types.String `tfsdk:"email"`
}

func (d *currentUserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_current_user"
}

func (d *currentUserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The identity of the credentials this provider is configured with (\"GET /api/1.0/user\" " +
			"- see DESIGN.md §5/§7). Its \"name\" is the developer_id used elsewhere in this provider, such " +
			"as developerhub_application.developer_id, exactly as SAP's own documentation describes " +
			"obtaining it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Same value as \"name\" (present for Terraform Plugin Framework data source convention).",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The developer id (\"Name\" field in the API response).",
			},
			"first_name": schema.StringAttribute{
				Computed: true,
			},
			"last_name": schema.StringAttribute{
				Computed: true,
			},
			"email": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *currentUserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data, ok := dataFromProviderData(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	client, ok := requireClient(data, "data source", &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = client
}

func (d *currentUserDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	user, err := d.client.GetCurrentUser(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read the current Developer Hub user", diagnosticDetail(err))
		return
	}

	state := currentUserModel{
		ID:        types.StringValue(user.Name),
		Name:      types.StringValue(user.Name),
		FirstName: types.StringValue(user.FirstName),
		LastName:  types.StringValue(user.LastName),
		Email:     types.StringValue(user.Email),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
