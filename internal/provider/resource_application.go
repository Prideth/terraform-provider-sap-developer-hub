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
	_ resource.Resource                = &applicationResource{}
	_ resource.ResourceWithConfigure   = &applicationResource{}
	_ resource.ResourceWithImportState = &applicationResource{}
)

// NewApplicationResource is registered in provider.go's Resources().
func NewApplicationResource() resource.Resource {
	return &applicationResource{}
}

type applicationResource struct {
	client *developerhub.Client
}

type applicationAttributeModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type applicationModel struct {
	ID          types.String                `tfsdk:"id"`
	Title       types.String                `tfsdk:"title"`
	Version     types.String                `tfsdk:"version"`
	DeveloperID types.String                `tfsdk:"developer_id"`
	Attribute   []applicationAttributeModel `tfsdk:"attribute"`
}

func (r *applicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *applicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Developer Hub application (\"APIMgmt.Applications\" - see DESIGN.md §5/§6). An " +
			"application is what an application developer subscribes to products with; use " +
			"developerhub_product_subscription to attach products to it.\n\n" +
			"Every application also has a generated app key/secret in Developer Hub, but no public API " +
			"documentation was found showing their field names, so this resource does not expose them - " +
			"see DESIGN.md §12.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The SAP-assigned application id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "The application's display title.",
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "The application entity version, as assigned by SAP.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"developer_id": schema.StringAttribute{
				Optional: true,
				Description: "The Developer Hub developer id (registered user id) this application " +
					"belongs to. Use the developerhub_current_user or developerhub_registered_users data " +
					"source to look this up rather than copying it from the UI by hand.",
			},
		},
		Blocks: map[string]schema.Block{
			"attribute": schema.ListNestedBlock{
				Description: "A custom attribute on the application (\"APIMgmt.Attributes\" - see " +
					"DESIGN.md §5/§6). Up to 18 are permitted, and each name is limited to 255 characters " +
					"and each value to 1024 characters, per SAP's documented limits.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The attribute name. Cannot be changed once created; delete and re-add instead.",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The attribute value.",
						},
					},
				},
			},
		},
	}
}

func (r *applicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *applicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan applicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateApplication(ctx, developerhub.Application{
		Title:       plan.Title.ValueString(),
		DeveloperID: plan.DeveloperID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to create Developer Hub application %q", plan.Title.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	for _, attr := range plan.Attribute {
		if err := r.client.CreateApplicationAttribute(ctx, created.ID, attr.Name.ValueString(), attr.Value.ValueString()); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to set attribute %q on Developer Hub application %q", attr.Name.ValueString(), created.ID),
				diagnosticDetail(err),
			)
			return
		}
	}

	plan.ID = types.StringValue(created.ID)
	plan.Version = types.StringValue(created.Version)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *applicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state applicationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetApplication(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Developer Hub application %q", state.ID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	attrs, err := r.client.ListApplicationAttributes(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read attributes of Developer Hub application %q", state.ID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	state.Title = types.StringValue(app.Title)
	state.Version = types.StringValue(app.Version)
	if app.DeveloperID != "" {
		state.DeveloperID = types.StringValue(app.DeveloperID)
	}
	state.Attribute = make([]applicationAttributeModel, 0, len(attrs))
	for _, attr := range attrs {
		state.Attribute = append(state.Attribute, applicationAttributeModel{
			Name:  types.StringValue(attr.Name),
			Value: types.StringValue(attr.Value),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *applicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state applicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Title.ValueString() != state.Title.ValueString() || plan.DeveloperID.ValueString() != state.DeveloperID.ValueString() {
		err := r.client.UpdateApplication(ctx, state.ID.ValueString(), developerhub.Application{
			Title:       plan.Title.ValueString(),
			DeveloperID: plan.DeveloperID.ValueString(),
			Version:     state.Version.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to update Developer Hub application %q", state.ID.ValueString()),
				diagnosticDetail(err),
			)
			return
		}
	}

	if err := r.reconcileAttributes(ctx, state.ID.ValueString(), state.Attribute, plan.Attribute); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update attributes of Developer Hub application %q", state.ID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	plan.ID = state.ID
	plan.Version = state.Version
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// reconcileAttributes diffs the previous and desired attribute lists by
// name and issues the minimal set of create/update/delete calls, since the
// Developer Hub API has no bulk "replace all attributes" operation.
func (r *applicationResource) reconcileAttributes(ctx context.Context, applicationID string, previous, desired []applicationAttributeModel) error {
	previousByName := make(map[string]string, len(previous))
	for _, attr := range previous {
		previousByName[attr.Name.ValueString()] = attr.Value.ValueString()
	}

	desiredByName := make(map[string]string, len(desired))
	for _, attr := range desired {
		desiredByName[attr.Name.ValueString()] = attr.Value.ValueString()
	}

	for name, value := range desiredByName {
		oldValue, existed := previousByName[name]
		switch {
		case !existed:
			if err := r.client.CreateApplicationAttribute(ctx, applicationID, name, value); err != nil {
				return err
			}
		case oldValue != value:
			if err := r.client.UpdateApplicationAttribute(ctx, applicationID, name, value); err != nil {
				return err
			}
		}
	}

	for name := range previousByName {
		if _, stillWanted := desiredByName[name]; !stillWanted {
			if err := r.client.DeleteApplicationAttribute(ctx, applicationID, name); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *applicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state applicationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteApplication(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Developer Hub application %q", state.ID.ValueString()),
			diagnosticDetail(err),
		)
	}
}

func (r *applicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
