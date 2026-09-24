package developerhub

import (
	"context"
	"net/http"
	"testing"
)

func TestGetCurrentUser(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/1.0/user" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"Name":"jdoe","FirstName":"Jane","LastName":"Doe","LoggedOut":false,"Email":"jane.doe@example.com"}]`))
	})
	defer server.Close()

	user, err := client.GetCurrentUser(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentUser: %v", err)
	}
	if user.Name != "jdoe" || user.Email != "jane.doe@example.com" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestGetCurrentUser_EmptyResponse(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})
	defer server.Close()

	if _, err := client.GetCurrentUser(context.Background()); err == nil {
		t.Fatal("expected an error for an empty response")
	}
}

func TestListRegisteredUsers(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/1.0/registrations" || r.URL.RawQuery != "type=registered" {
			t.Fatalf("unexpected request %q?%q", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"userId":"jdoe","firstName":"Jane","lastName":"Doe","emailId":"jane.doe@example.com","country":"DE","autoReLogin":false,"rolesAccess":[{"status":"registered","role":"API_ApplicationDeveloper"}]}]`))
	})
	defer server.Close()

	users, err := client.ListRegisteredUsers(context.Background())
	if err != nil {
		t.Fatalf("ListRegisteredUsers: %v", err)
	}
	if len(users) != 1 || users[0].UserID != "jdoe" || len(users[0].RolesAccess) != 1 {
		t.Fatalf("unexpected users: %+v", users)
	}
}
