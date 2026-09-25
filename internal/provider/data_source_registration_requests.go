package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/developerhub"
)

var (
	_ datasource.DataSource              = &registrationRequestsDataSource{}
	_ datasource.DataSourceWithConfigure = &registrationRequestsDataSource{}
)

// NewRegistrationRequestsDataSource is registered in provider.go's DataSources().
func NewRegistrationRequestsDataSource() datasource.DataSource {
	return &registrationRequestsDataSource{}
}

type registrationRequestsDataSource struct {
	client *developerhub.Client
}

type registrationRequestModel struct {
	UserID    types.String `tfsdk:"user_id"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	Email     types.String `tfsdk:"email"`
	Country   types.String `tfsdk:"country"`
}

type registrationRequestsModel struct {
	ID      types.String               `tfsdk:"id"`
	Request []registrationRequestModel `tfsdk:"request"`
}

func (d *registrationRequestsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registration_requests"
}

func (d *registrationRequestsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Users whose request to register as an application developer still awaits an " +
			"administrator decision (\"GET /api/1.0/registrations?type=pending\" - see DESIGN.md §5/§7).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Fixed identifier (\"registration_requests\") for Terraform Plugin Framework data source convention.",
			},
			"request": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The pending registration requests.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id":    schema.StringAttribute{Computed: true},
						"first_name": schema.StringAttribute{Computed: true},
						"last_name":  schema.StringAttribute{Computed: true},
						"email":      schema.StringAttribute{Computed: true},
						"country":    schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *registrationRequestsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *registrationRequestsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	pending, err := d.client.ListPendingRegistrations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list pending Developer Hub registration requests", diagnosticDetail(err))
		return
	}

	state := registrationRequestsModel{
		ID:      types.StringValue("registration_requests"),
		Request: make([]registrationRequestModel, 0, len(pending)),
	}
	for _, u := range pending {
		state.Request = append(state.Request, registrationRequestModel{
			UserID:    types.StringValue(u.UserID),
			FirstName: types.StringValue(u.FirstName),
			LastName:  types.StringValue(u.LastName),
			Email:     types.StringValue(u.EmailID),
			Country:   types.StringValue(u.Country),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
