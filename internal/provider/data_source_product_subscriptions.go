package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/developerhub"
)

var (
	_ datasource.DataSource              = &productSubscriptionsDataSource{}
	_ datasource.DataSourceWithConfigure = &productSubscriptionsDataSource{}
)

// NewProductSubscriptionsDataSource is registered in provider.go's DataSources().
func NewProductSubscriptionsDataSource() datasource.DataSource {
	return &productSubscriptionsDataSource{}
}

type productSubscriptionsDataSource struct {
	client *developerhub.Client
}

type productSubscriptionSummaryModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.String `tfsdk:"application_id"`
	ProductName   types.String `tfsdk:"product_name"`
	DeveloperID   types.String `tfsdk:"developer_id"`
	IsSubscribed  types.Bool   `tfsdk:"is_subscribed"`
	Status        types.String `tfsdk:"status"`
}

type productSubscriptionsModel struct {
	ID            types.String                      `tfsdk:"id"`
	ApplicationID types.String                      `tfsdk:"application_id"`
	ProductName   types.String                      `tfsdk:"product_name"`
	Subscription  []productSubscriptionSummaryModel `tfsdk:"subscription"`
}

func (d *productSubscriptionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_subscriptions"
}

func (d *productSubscriptionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists Developer Hub product subscriptions (the \"APIMgmt.Subscriptions\" entity set - " +
			"see DESIGN.md §5/§7/§9), optionally narrowed to one application or one product. Useful to see " +
			"subscriptions created outside Terraform, for example by developers in the Developer Hub UI, " +
			"and their current approval status.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Fixed identifier (\"product_subscriptions\") for Terraform Plugin Framework data source convention.",
			},
			"application_id": schema.StringAttribute{
				Optional:    true,
				Description: "If set, only subscriptions of this application are returned.",
			},
			"product_name": schema.StringAttribute{
				Optional:    true,
				Description: "If set, only subscriptions to this product (technical name) are returned.",
			},
			"subscription": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The subscriptions, in the order the API returns them.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":             schema.StringAttribute{Computed: true, Description: "The SAP-assigned subscription id."},
						"application_id": schema.StringAttribute{Computed: true},
						"product_name":   schema.StringAttribute{Computed: true},
						"developer_id":   schema.StringAttribute{Computed: true},
						"is_subscribed":  schema.BoolAttribute{Computed: true},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "SAP's own subscription status, passed through unvalidated (DESIGN.md §9).",
						},
					},
				},
			},
		},
	}
}

func (d *productSubscriptionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *productSubscriptionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config productSubscriptionsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subs, err := d.client.ListSubscriptions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Developer Hub product subscriptions", diagnosticDetail(err))
		return
	}

	state := productSubscriptionsModel{
		ID:            types.StringValue("product_subscriptions"),
		ApplicationID: config.ApplicationID,
		ProductName:   config.ProductName,
		Subscription:  filterSubscriptions(subs, config.ApplicationID.ValueString(), config.ProductName.ValueString()),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func filterSubscriptions(subs []developerhub.Subscription, applicationID, productName string) []productSubscriptionSummaryModel {
	result := make([]productSubscriptionSummaryModel, 0, len(subs))
	for _, sub := range subs {
		if applicationID != "" && sub.AppID != applicationID {
			continue
		}
		if productName != "" && sub.ProductID != productName {
			continue
		}
		result = append(result, productSubscriptionSummaryModel{
			ID:            types.StringValue(sub.ID),
			ApplicationID: types.StringValue(sub.AppID),
			ProductName:   types.StringValue(sub.ProductID),
			DeveloperID:   types.StringValue(sub.DeveloperID),
			IsSubscribed:  types.BoolValue(sub.IsSubscribed),
			Status:        types.StringValue(sub.Status),
		})
	}
	return result
}
