package developerhub

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestRegisterDeveloper(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/1.0/registrations" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body DeveloperRegistration
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		if body.UserID != "jdoe" || body.EmailID != "jane@example.com" {
			t.Fatalf("unexpected registration: %+v", body)
		}
		if len(body.RolesAccess) != 1 || body.RolesAccess[0].Role != ApplicationDeveloperRole {
			t.Fatalf("expected the application developer role to be requested, got %+v", body.RolesAccess)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"userId":"jdoe","emailId":"jane@example.com","rolesAccess":[{"status":"registered","role":"API_ApplicationDeveloper"}]}`))
	})
	defer server.Close()

	created, err := client.RegisterDeveloper(context.Background(), DeveloperRegistration{
		UserID: "jdoe", EmailID: "jane@example.com", FirstName: "Jane", LastName: "Doe",
	})
	if err != nil {
		t.Fatalf("RegisterDeveloper: %v", err)
	}
	if created.DeveloperStatus() != RegistrationStatusRegistered {
		t.Fatalf("expected status registered, got %q", created.DeveloperStatus())
	}
}

func TestSetRegistrationStatus(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.EscapedPath() != "/api/1.0/registrations/j%20doe" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.EscapedPath())
		}
		var body developerRolesUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		if len(body.RolesAccess) != 1 || body.RolesAccess[0].Status != RegistrationStatusRevoked ||
			body.RolesAccess[0].Role != ApplicationDeveloperRole || body.RolesAccess[0].ResponseReason != "left the team" {
			t.Fatalf("unexpected roles update: %+v", body)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	if err := client.SetRegistrationStatus(context.Background(), "j doe", RegistrationStatusRevoked, "left the team"); err != nil {
		t.Fatalf("SetRegistrationStatus: %v", err)
	}
}

func TestListPendingRegistrations(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/1.0/registrations" || r.URL.RawQuery != "type=pending" {
			t.Fatalf("unexpected request %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"userId":"new-dev","emailId":"new@example.com","rolesAccess":[{"status":"pending","role":"API_ApplicationDeveloper"}]}]`))
	})
	defer server.Close()

	pending, err := client.ListPendingRegistrations(context.Background())
	if err != nil {
		t.Fatalf("ListPendingRegistrations: %v", err)
	}
	if len(pending) != 1 || pending[0].UserID != "new-dev" {
		t.Fatalf("unexpected pending registrations: %+v", pending)
	}
}

func TestDeveloperStatus_NoDeveloperRole(t *testing.T) {
	u := RegisteredUser{RolesAccess: []RoleAccess{{Role: "SomethingElse", Status: "registered"}}}
	if u.DeveloperStatus() != "" {
		t.Fatalf("expected no developer status, got %q", u.DeveloperStatus())
	}
}

func TestListProducts(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != productsPath {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"d":{"results":[{"name":"Sales_API","title":"Sales API","version":"1","shortText":"Sales","published_by":"admin"}]}}`))
	})
	defer server.Close()

	products, err := client.ListProducts(context.Background())
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(products) != 1 || products[0].Name != "Sales_API" || products[0].ShortText != "Sales" {
		t.Fatalf("unexpected products: %+v", products)
	}
}
