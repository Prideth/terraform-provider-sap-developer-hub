package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/apierror"
	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/developerhub"
)

var (
	_ resource.Resource                = &developerResource{}
	_ resource.ResourceWithConfigure   = &developerResource{}
	_ resource.ResourceWithImportState = &developerResource{}
)

// NewDeveloperResource is registered in provider.go's Resources().
func NewDeveloperResource() resource.Resource {
	return &developerResource{}
}

type developerResource struct {
	client *developerhub.Client
}

type developerModel struct {
	ID               types.String `tfsdk:"id"`
	UserID           types.String `tfsdk:"user_id"`
	Email            types.String `tfsdk:"email"`
	FirstName        types.String `tfsdk:"first_name"`
	LastName         types.String `tfsdk:"last_name"`
	Country          types.String `tfsdk:"country"`
	Status           types.String `tfsdk:"status"`
	RevocationReason types.String `tfsdk:"revocation_reason"`
}

func (r *developerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_developer"
}

func (r *developerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Registers a user as an application developer on Developer Hub (the \"Add User\" flow, " +
			"\"POST /api/1.0/registrations\" performed as an administrator - see DESIGN.md §5/§6). " +
			"Destroying the resource revokes the user's Developer Hub access " +
			"(\"PUT /api/1.0/registrations/{id}\" with status revoked); it does not delete the user from any " +
			"identity provider.\n\n" +
			"This manages Developer Hub's own registration state only. It does not assign SAP BTP role " +
			"collections - use the SAP BTP provider for those. Note that SAP documents that accepting a user " +
			"in Developer Hub adds them to the subaccount's identity providers with the application developer " +
			"role; that is Developer Hub's own behavior, not something this provider does separately.\n\n" +
			"The registration API has no endpoint to change a user's name, e-mail or country, so changing any " +
			"of them replaces the registration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Same as user_id.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"user_id": schema.StringAttribute{
				Required: true,
				Description: "The user's id in the subaccount's identity provider; becomes the developer_id used by " +
					"developerhub_application.",
				PlanModifiers: replace,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"email": schema.StringAttribute{
				Required:      true,
				Description:   "The user's e-mail address.",
				PlanModifiers: replace,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"first_name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: replace,
			},
			"last_name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: replace,
			},
			"country": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: replace,
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The registration status Developer Hub reports for the application developer role.",
			},
			"revocation_reason": schema.StringAttribute{
				Optional: true,
				Description: "Reason sent with the revocation when this resource is destroyed; Developer Hub " +
					"notifies the user by e-mail. Changing it only updates state.",
			},
		},
	}
}

func (r *developerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *developerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan developerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	userID := plan.UserID.ValueString()

	created, err := r.client.RegisterDeveloper(ctx, developerhub.DeveloperRegistration{
		UserID:    userID,
		EmailID:   plan.Email.ValueString(),
		FirstName: plan.FirstName.ValueString(),
		LastName:  plan.LastName.ValueString(),
		Country:   plan.Country.ValueString(),
	})
	status := ""
	switch {
	case err == nil:
		status = created.DeveloperStatus()
	case isConflict(err):
		// The user is already known to Developer Hub, for example because
		// their access was revoked earlier: grant it again through the
		// documented status update instead.
		if err := r.client.SetRegistrationStatus(ctx, userID, developerhub.RegistrationStatusRegistered, ""); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Unable to register Developer Hub developer %q", userID), diagnosticDetail(err))
			return
		}
		status = developerhub.RegistrationStatusRegistered
	default:
		resp.Diagnostics.AddError(fmt.Sprintf("Unable to register Developer Hub developer %q", userID), diagnosticDetail(err))
		return
	}
	if status == "" {
		status = developerhub.RegistrationStatusRegistered
	}

	plan.ID = types.StringValue(userID)
	plan.Status = types.StringValue(status)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *developerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state developerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	userID := state.UserID.ValueString()

	users, err := r.client.ListRegisteredUsers(ctx)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Unable to read Developer Hub developer %q", userID), diagnosticDetail(err))
		return
	}
	user, found := findRegisteredUser(users, userID)
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	switch user.DeveloperStatus() {
	case developerhub.RegistrationStatusRevoked, developerhub.RegistrationStatusRejected:
		// No longer a developer: plan a new registration.
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(user.UserID)
	state.Email = types.StringValue(user.EmailID)
	state.FirstName = types.StringValue(user.FirstName)
	state.LastName = types.StringValue(user.LastName)
	state.Country = optionalString(state.Country, user.Country)
	state.Status = types.StringValue(user.DeveloperStatus())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is only reached for revocation_reason, the one attribute without
// RequiresReplace; it lives in state only, so there is nothing to call.
func (r *developerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state developerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	plan.Status = state.Status
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *developerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state developerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	userID := state.UserID.ValueString()
	err := r.client.SetRegistrationStatus(ctx, userID, developerhub.RegistrationStatusRevoked, state.RevocationReason.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Unable to revoke Developer Hub developer %q", userID), diagnosticDetail(err))
	}
}

func (r *developerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("user_id"), req, resp)
}

func findRegisteredUser(users []developerhub.RegisteredUser, userID string) (developerhub.RegisteredUser, bool) {
	for _, u := range users {
		if u.UserID == userID {
			return u, true
		}
	}
	return developerhub.RegisteredUser{}, false
}

// isConflict reports whether err represents a 409 from the Developer Hub API.
func isConflict(err error) bool {
	var apiErr *apierror.Error
	return errors.As(err, &apiErr) && apiErr.IsConflict()
}
