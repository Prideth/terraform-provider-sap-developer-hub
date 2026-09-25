package developerhub

import (
	"context"
	"fmt"
	"net/url"
)

// CurrentUser is the identity of the credentials used to authenticate,
// exactly as documented in accessing-developer-hub-apis-programmatically-dabee6e.md
// (DESIGN.md §5/§11), including its "Name" field, which the SAP
// documentation states doubles as the developer_id used elsewhere in the
// API (e.g. developerhub_application.developer_id).
type CurrentUser struct {
	Name      string `json:"Name"`
	FirstName string `json:"FirstName"`
	LastName  string `json:"LastName"`
	LoggedOut bool   `json:"LoggedOut"`
	Email     string `json:"Email"`
}

// GetCurrentUser calls "GET /api/1.0/user", which SAP documents as
// returning a single-element JSON array.
func (c *Client) GetCurrentUser(ctx context.Context) (*CurrentUser, error) {
	var users []CurrentUser
	if err := c.get(ctx, "/api/1.0/user", &users); err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("developer hub API returned no current user")
	}
	return &users[0], nil
}

// ApplicationDeveloperRole is the Developer Hub role every registration
// carries; it is the only role value the DevPortal_RegisteringUsers_CF
// specification (api-specs/) documents.
const ApplicationDeveloperRole = "API_ApplicationDeveloper"

// Registration statuses, from the specification's rolesAccess.status enums.
const (
	RegistrationStatusRegistered = "registered"
	RegistrationStatusRevoked    = "revoked"
	RegistrationStatusRejected   = "rejected"
)

// RoleAccess is one role entry of a registration's rolesAccess list.
type RoleAccess struct {
	Status         string `json:"status,omitempty"`
	Role           string `json:"role"`
	ResponseReason string `json:"responseReason,omitempty"`
	RequestReason  string `json:"requestReason,omitempty"`
}

// RegisteredUser is one entry of "GET /api/1.0/registrations?type=...":
// "developerEntity" in the DevPortal_RegisteringUsers_CF specification. Its
// userId is the same value used as developer_id elsewhere in the API.
type RegisteredUser struct {
	UserID      string       `json:"userId"`
	FirstName   string       `json:"firstName"`
	LastName    string       `json:"lastName"`
	EmailID     string       `json:"emailId"`
	Country     string       `json:"country"`
	AutoReLogin bool         `json:"autoReLogin"`
	RolesAccess []RoleAccess `json:"rolesAccess"`
}

// DeveloperStatus returns the status of the user's application developer
// role, or "" if the user has none.
func (u RegisteredUser) DeveloperStatus() string {
	for _, r := range u.RolesAccess {
		if r.Role == ApplicationDeveloperRole {
			return r.Status
		}
	}
	return ""
}

// ListRegisteredUsers calls "GET /api/1.0/registrations?type=registered".
func (c *Client) ListRegisteredUsers(ctx context.Context) ([]RegisteredUser, error) {
	return c.listRegistrations(ctx, "registered")
}

// ListPendingRegistrations calls "GET /api/1.0/registrations?type=pending":
// users whose registration request still awaits an administrator decision.
func (c *Client) ListPendingRegistrations(ctx context.Context) ([]RegisteredUser, error) {
	return c.listRegistrations(ctx, "pending")
}

func (c *Client) listRegistrations(ctx context.Context, kind string) ([]RegisteredUser, error) {
	var collection odataCollection[RegisteredUser]
	if err := c.get(ctx, "/api/1.0/registrations?type="+kind, &collection); err != nil {
		return nil, err
	}
	return collection.Items, nil
}

// DeveloperRegistration is the specification's "developerPostEntity".
type DeveloperRegistration struct {
	UserID      string       `json:"userId"`
	EmailID     string       `json:"emailId"`
	FirstName   string       `json:"firstName"`
	LastName    string       `json:"lastName"`
	Country     string       `json:"country,omitempty"`
	RolesAccess []RoleAccess `json:"rolesAccess"`
}

// RegisterDeveloper calls "POST /api/1.0/registrations". Per the
// specification, when an administrator performs it the user is registered
// directly rather than raising an access request.
func (c *Client) RegisterDeveloper(ctx context.Context, reg DeveloperRegistration) (*RegisteredUser, error) {
	if len(reg.RolesAccess) == 0 {
		reg.RolesAccess = []RoleAccess{{Role: ApplicationDeveloperRole}}
	}
	var created RegisteredUser
	if err := c.post(ctx, "/api/1.0/registrations", reg, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

type developerRolesUpdate struct {
	RolesAccess []RoleAccess `json:"rolesAccess"`
}

// SetRegistrationStatus calls "PUT /api/1.0/registrations/{developerid}" to
// accept (registered), reject (rejected) or revoke (revoked) a user's
// application developer access.
func (c *Client) SetRegistrationStatus(ctx context.Context, userID, status, reason string) error {
	body := developerRolesUpdate{RolesAccess: []RoleAccess{{
		Status:         status,
		Role:           ApplicationDeveloperRole,
		ResponseReason: reason,
	}}}
	return c.put(ctx, "/api/1.0/registrations/"+url.PathEscape(userID), body)
}
