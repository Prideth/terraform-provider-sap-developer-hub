package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/developerhub"
)

var (
	_ datasource.DataSource              = &registeredUsersDataSource{}
	_ datasource.DataSourceWithConfigure = &registeredUsersDataSource{}
)

// NewRegisteredUsersDataSource is registered in provider.go's DataSources().
func NewRegisteredUsersDataSource() datasource.DataSource {
	return &registeredUsersDataSource{}
}

type registeredUsersDataSource struct {
	client *developerhub.Client
}

type registeredUserRoleModel struct {
	Status types.String `tfsdk:"status"`
	Role   types.String `tfsdk:"role"`
}

type registeredUserModel struct {
	UserID    types.String              `tfsdk:"user_id"`
	FirstName types.String              `tfsdk:"first_name"`
	LastName  types.String              `tfsdk:"last_name"`
	Email     types.String              `tfsdk:"email"`
	Country   types.String              `tfsdk:"country"`
	Role      []registeredUserRoleModel `tfsdk:"role"`
}

type registeredUsersModel struct {
	ID   types.String          `tfsdk:"id"`
	User []registeredUserModel `tfsdk:"user"`
}

func (d *registeredUsersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registered_users"
}

func (d *registeredUsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Registered Developer Hub developers (\"GET /api/1.0/registrations?type=registered\" - " +
			"see DESIGN.md §5/§7). Use this to look up a user_id (== developer_id) for " +
			"developerhub_application.developer_id rather than copying it from the Developer Hub UI by hand.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Fixed identifier (\"registered_users\") for Terraform Plugin Framework data source convention.",
			},
			"user": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The registered developers.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id": schema.StringAttribute{
							Computed:    true,
							Description: "The developer id, usable as developerhub_application.developer_id.",
						},
						"first_name": schema.StringAttribute{Computed: true},
						"last_name":  schema.StringAttribute{Computed: true},
						"email":      schema.StringAttribute{Computed: true},
						"country":    schema.StringAttribute{Computed: true},
						"role": schema.ListNestedAttribute{
							Computed:    true,
							Description: "The developer's role assignments and their status.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"status": schema.StringAttribute{Computed: true},
									"role":   schema.StringAttribute{Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *registeredUsersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *registeredUsersDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	users, err := d.client.ListRegisteredUsers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list registered Developer Hub users", diagnosticDetail(err))
		return
	}

	state := registeredUsersModel{
		ID:   types.StringValue("registered_users"),
		User: make([]registeredUserModel, 0, len(users)),
	}
	for _, u := range users {
		roles := make([]registeredUserRoleModel, 0, len(u.RolesAccess))
		for _, role := range u.RolesAccess {
			roles = append(roles, registeredUserRoleModel{
				Status: types.StringValue(role.Status),
				Role:   types.StringValue(role.Role),
			})
		}
		state.User = append(state.User, registeredUserModel{
			UserID:    types.StringValue(u.UserID),
			FirstName: types.StringValue(u.FirstName),
			LastName:  types.StringValue(u.LastName),
			Email:     types.StringValue(u.EmailID),
			Country:   types.StringValue(u.Country),
			Role:      roles,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
