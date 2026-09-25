package developerhub

import (
	"context"
	"fmt"
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

// RoleAccess is one role entry of a RegisteredUser's rolesAccess list.
type RoleAccess struct {
	Status string `json:"status"`
	Role   string `json:"role"`
}

// RegisteredUser is one entry of the "GET /api/1.0/registrations?type=registered"
// response documented in accessing-developer-hub-apis-programmatically-dabee6e.md
// (DESIGN.md §5). Its userId is the same value used as developer_id
// elsewhere in the API.
type RegisteredUser struct {
	UserID      string       `json:"userId"`
	FirstName   string       `json:"firstName"`
	LastName    string       `json:"lastName"`
	EmailID     string       `json:"emailId"`
	Country     string       `json:"country"`
	AutoReLogin bool         `json:"autoReLogin"`
	RolesAccess []RoleAccess `json:"rolesAccess"`
}

// ListRegisteredUsers calls "GET /api/1.0/registrations?type=registered".
func (c *Client) ListRegisteredUsers(ctx context.Context) ([]RegisteredUser, error) {
	var collection odataCollection[RegisteredUser]
	if err := c.get(ctx, "/api/1.0/registrations?type=registered", &collection); err != nil {
		return nil, err
	}
	return collection.Items, nil
}
