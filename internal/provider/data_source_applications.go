package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/developerhub"
)

var (
	_ datasource.DataSource              = &applicationsDataSource{}
	_ datasource.DataSourceWithConfigure = &applicationsDataSource{}
)

// NewApplicationsDataSource is registered in provider.go's DataSources().
func NewApplicationsDataSource() datasource.DataSource {
	return &applicationsDataSource{}
}

type applicationsDataSource struct {
	client *developerhub.Client
}

type applicationSummaryModel struct {
	ID          types.String `tfsdk:"id"`
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	ShortText   types.String `tfsdk:"short_text"`
	CallbackURL types.String `tfsdk:"callback_url"`
	DeveloperID types.String `tfsdk:"developer_id"`
	Version     types.String `tfsdk:"version"`
}

type applicationsModel struct {
	ID          types.String              `tfsdk:"id"`
	DeveloperID types.String              `tfsdk:"developer_id"`
	Application []applicationSummaryModel `tfsdk:"application"`
}

func (d *applicationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_applications"
}

func (d *applicationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists Developer Hub applications (\"GET APIMgmt.Applications\" - see DESIGN.md §5/§7), " +
			"for example to reference applications that are not managed by this Terraform configuration. " +
			"SAP's collection read does not return application keys or secrets, so this data source " +
			"cannot expose them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Fixed identifier (\"applications\") for Terraform Plugin Framework data source convention.",
			},
			"developer_id": schema.StringAttribute{
				Optional:    true,
				Description: "If set, only applications belonging to this developer id are returned.",
			},
			"application": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The applications, in the order the API returns them.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":           schema.StringAttribute{Computed: true, Description: "The SAP-assigned application id."},
						"title":        schema.StringAttribute{Computed: true},
						"description":  schema.StringAttribute{Computed: true},
						"short_text":   schema.StringAttribute{Computed: true},
						"callback_url": schema.StringAttribute{Computed: true},
						"developer_id": schema.StringAttribute{Computed: true},
						"version":      schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *applicationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *applicationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config applicationsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apps, err := d.client.ListApplications(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Developer Hub applications", diagnosticDetail(err))
		return
	}

	state := applicationsModel{
		ID:          types.StringValue("applications"),
		DeveloperID: config.DeveloperID,
		Application: filterApplications(apps, config.DeveloperID.ValueString()),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func filterApplications(apps []developerhub.Application, developerID string) []applicationSummaryModel {
	result := make([]applicationSummaryModel, 0, len(apps))
	for _, app := range apps {
		if developerID != "" && app.DeveloperID != developerID {
			continue
		}
		result = append(result, applicationSummaryModel{
			ID:          types.StringValue(app.ID),
			Title:       types.StringValue(app.Title),
			Description: types.StringValue(app.Description),
			ShortText:   types.StringValue(app.ShortText),
			CallbackURL: types.StringValue(app.CallbackURL),
			DeveloperID: types.StringValue(app.DeveloperID),
			Version:     types.StringValue(app.Version),
		})
	}
	return result
}
