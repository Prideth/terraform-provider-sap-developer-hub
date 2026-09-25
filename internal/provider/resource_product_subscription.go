package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/developerhub"
)

var (
	_ resource.Resource                = &productSubscriptionResource{}
	_ resource.ResourceWithConfigure   = &productSubscriptionResource{}
	_ resource.ResourceWithImportState = &productSubscriptionResource{}
)

// NewProductSubscriptionResource is registered in provider.go's Resources().
func NewProductSubscriptionResource() resource.Resource {
	return &productSubscriptionResource{}
}

type productSubscriptionResource struct {
	client *developerhub.Client
}

type productSubscriptionModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.String `tfsdk:"application_id"`
	ProductName   types.String `tfsdk:"product_name"`
	Status        types.String `tfsdk:"status"`
}

func (r *productSubscriptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_subscription"
}

func (r *productSubscriptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Subscribes a Developer Hub application to a product (the top-level \"APIMgmt.Subscriptions\" " +
			"OData entity - see DESIGN.md §5/§6/§9). This does not create or manage the product itself - " +
			"products are authored and published from SAP Integration Suite / API Portal and are managed by " +
			"terraform-provider-integration-suite; this resource only references a product by its existing " +
			"technical name.\n\n" +
			"Whether a subscription needs external approval, and how that approval happens, depends on the " +
			"tenant's Subscription Governance setting, which has no documented public API and so cannot be " +
			"read or changed by this provider (DESIGN.md §12) - the subscription is created here exactly as " +
			"SAP's own governance configuration for the tenant handles it, and its resulting approval state " +
			"is exposed read-only via the status attribute. If the tenant uses External Governance and the " +
			"request is rejected, SAP deletes the subscription itself; this resource detects that on its " +
			"next refresh and removes it from state, the same as if it had been deleted any other way.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The SAP-assigned subscription id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "The id of the developerhub_application this subscription belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"product_name": schema.StringAttribute{
				Required: true,
				Description: "The technical name of the product to subscribe to, as published from SAP " +
					"Integration Suite / API Portal (not this provider). Changing it updates the " +
					"subscription in place.",
			},
			"status": schema.StringAttribute{
				Computed: true,
				Description: "SAP's own subscription status (for example, a pending-approval state under " +
					"External Governance). Its exact set of possible values is not documented publicly, so " +
					"this provider passes it through as an opaque string rather than validating it - see " +
					"DESIGN.md §9.",
			},
		},
	}
}

func (r *productSubscriptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data, ok := dataFromProviderData(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	client, ok := requireClient(data, "resource", &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = client
}

func (r *productSubscriptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan productSubscriptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateSubscription(ctx, plan.ApplicationID.ValueString(), plan.ProductName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to subscribe application %q to product %q", plan.ApplicationID.ValueString(), plan.ProductName.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.Status = types.StringValue(created.Status)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *productSubscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state productSubscriptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sub, err := r.client.GetSubscription(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Developer Hub product subscription %q", state.ID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	if sub.AppID != "" {
		state.ApplicationID = types.StringValue(sub.AppID)
	}
	if sub.ProductID != "" {
		state.ProductName = types.StringValue(sub.ProductID)
	}
	state.Status = types.StringValue(sub.Status)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *productSubscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state productSubscriptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ProductName.ValueString() != state.ProductName.ValueString() {
		if err := r.client.UpdateSubscriptionProduct(ctx, state.ID.ValueString(), plan.ProductName.ValueString()); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to update Developer Hub product subscription %q", state.ID.ValueString()),
				diagnosticDetail(err),
			)
			return
		}
	}

	plan.ID = state.ID
	plan.Status = state.Status
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *productSubscriptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state productSubscriptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSubscription(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Developer Hub product subscription %q", state.ID.ValueString()),
			diagnosticDetail(err),
		)
	}
}

func (r *productSubscriptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
