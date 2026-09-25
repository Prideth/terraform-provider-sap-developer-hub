package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/developerhub"
)

var (
	_ datasource.DataSource              = &productsDataSource{}
	_ datasource.DataSourceWithConfigure = &productsDataSource{}
)

// NewProductsDataSource is registered in provider.go's DataSources().
func NewProductsDataSource() datasource.DataSource {
	return &productsDataSource{}
}

type productsDataSource struct {
	client *developerhub.Client
}

type productModel struct {
	Name        types.String `tfsdk:"name"`
	Title       types.String `tfsdk:"title"`
	Version     types.String `tfsdk:"version"`
	Vendor      types.String `tfsdk:"vendor"`
	Description types.String `tfsdk:"description"`
	ShortText   types.String `tfsdk:"short_text"`
	PublishedAt types.String `tfsdk:"published_at"`
	PublishedBy types.String `tfsdk:"published_by"`
}

type productsModel struct {
	ID      types.String   `tfsdk:"id"`
	Name    types.String   `tfsdk:"name"`
	Product []productModel `tfsdk:"product"`
}

func (d *productsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_products"
}

func (d *productsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the products published to the Developer Hub catalog (\"APIMgmt.APIProducts\" - " +
			"see DESIGN.md §5/§7), so a product's technical name can be looked up instead of hard-coded, for " +
			"example for developerhub_product_subscription.product_name. Read-only: products are authored and " +
			"published in SAP Integration Suite, not by this provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Fixed identifier (\"products\") for Terraform Plugin Framework data source convention.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "If set, only the product with this technical name is returned.",
			},
			"product": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The published products, in the order the API returns them.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":         schema.StringAttribute{Computed: true, Description: "The product's technical name (its key)."},
						"title":        schema.StringAttribute{Computed: true},
						"version":      schema.StringAttribute{Computed: true},
						"vendor":       schema.StringAttribute{Computed: true},
						"description":  schema.StringAttribute{Computed: true},
						"short_text":   schema.StringAttribute{Computed: true},
						"published_at": schema.StringAttribute{Computed: true, Description: "When the product was published, as returned by SAP."},
						"published_by": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *productsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *productsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config productsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	products, err := d.client.ListProducts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Developer Hub products", diagnosticDetail(err))
		return
	}

	state := productsModel{
		ID:      types.StringValue("products"),
		Name:    config.Name,
		Product: filterProducts(products, config.Name.ValueString()),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func filterProducts(products []developerhub.Product, name string) []productModel {
	result := make([]productModel, 0, len(products))
	for _, p := range products {
		if name != "" && p.Name != name {
			continue
		}
		result = append(result, productModel{
			Name:        types.StringValue(p.Name),
			Title:       types.StringValue(p.Title),
			Version:     types.StringValue(p.Version),
			Vendor:      types.StringValue(p.Vendor),
			Description: types.StringValue(p.Description),
			ShortText:   types.StringValue(p.ShortText),
			PublishedAt: types.StringValue(p.PublishedAt),
			PublishedBy: types.StringValue(p.PublishedBy),
		})
	}
	return result
}
