package provider

import (
	"context"
	"fmt"
	"strings"

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
}

func (r *productSubscriptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_subscription"
}

func (r *productSubscriptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Subscribes a Developer Hub application to a product (\"ToSubscriptions\"/\"ToAPIProduct\" " +
			"on \"APIMgmt.Applications\" - see DESIGN.md §5/§6/§9). This does not create or manage the " +
			"product itself - products are authored and published from SAP Integration Suite / API Portal " +
			"and are managed by terraform-provider-integration-suite; this resource only references a " +
			"product by its existing technical name.\n\n" +
			"Whether a subscription needs external approval, and how that approval happens, depends on the " +
			"tenant's Subscription Governance setting, which has no documented public API and so cannot be " +
			"read or changed by this provider (DESIGN.md §12) - the subscription is created here exactly as " +
			"SAP's own governance configuration for the tenant handles it.",
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
					"Integration Suite / API Portal (not this provider). Changing it replaces the " +
					"subscription, since no update endpoint for retargeting an existing subscription is " +
					"documented.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *productSubscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state productSubscriptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sub, err := r.client.GetSubscription(ctx, state.ApplicationID.ValueString(), state.ID.ValueString())
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

	if sub.ProductName != "" {
		state.ProductName = types.StringValue(sub.ProductName)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only ever runs for attributes with no RequiresReplace plan
// modifier; both application_id and product_name have one, so this method
// is unreachable in practice. It is implemented defensively (copy plan to
// state) rather than left to panic, in case a future schema change removes
// one of those plan modifiers without updating this method to match.
func (r *productSubscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan productSubscriptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *productSubscriptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state productSubscriptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSubscription(ctx, state.ApplicationID.ValueString(), state.ID.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Developer Hub product subscription %q", state.ID.ValueString()),
			diagnosticDetail(err),
		)
	}
}

func (r *productSubscriptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	applicationID, subscriptionID, ok := splitCompositeID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected import identifier format",
			fmt.Sprintf("Expected \"<application_id>/<subscription_id>\", got %q.", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), applicationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), subscriptionID)...)
}

// splitCompositeID splits a "<a>/<b>" import identifier into its two parts.
func splitCompositeID(id string) (first, second string, ok bool) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
