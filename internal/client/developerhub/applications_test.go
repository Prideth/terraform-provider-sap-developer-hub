package developerhub

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/apierror"
	sapthttp "github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/http"
)

func testDevPortalClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	httpClient := sapthttp.New(sapthttp.Config{Transport: http.DefaultClient, UserAgent: "test-agent"})
	return New(httpClient, server.URL), server
}

func TestCreateApplication(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != applicationsPath {
			t.Fatalf("expected path %q, got %q", applicationsPath, r.URL.Path)
		}
		var body Application
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body.Title != "Sales App" {
			t.Fatalf("expected title %q, got %q", "Sales App", body.Title)
		}
		if body.ID != placeholderID {
			t.Fatalf("expected create request id to be the documented placeholder %q, got %q", placeholderID, body.ID)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Application{
			ID: "app-1", Title: body.Title, Version: "1", DeveloperID: body.DeveloperID,
			AppKey: "generated-key", AppSecret: "generated-secret",
		})
	})
	defer server.Close()

	created, err := client.CreateApplication(context.Background(), Application{Title: "Sales App", DeveloperID: "dev-1"})
	if err != nil {
		t.Fatalf("CreateApplication: %v", err)
	}
	if created.ID != "app-1" || created.AppKey != "generated-key" || created.AppSecret != "generated-secret" {
		t.Fatalf("unexpected created application: %+v", created)
	}
}

func TestGetApplication_NotFound(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":{"value":"Application not found"}}}`))
	})
	defer server.Close()

	_, err := client.GetApplication(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected an error")
	}

	var apiErr *apierror.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *apierror.Error, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Fatalf("expected IsNotFound() == true, got status %d", apiErr.StatusCode)
	}
	if apiErr.Code != "NOT_FOUND" || apiErr.Message != "Application not found" {
		t.Fatalf("unexpected parsed error: %+v", apiErr)
	}
}

func TestGetApplication_WrappedEnvelope(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"d":{"id":"app-1","title":"Sales App","version":"1"}}`))
	})
	defer server.Close()

	app, err := client.GetApplication(context.Background(), "app-1")
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if app.ID != "app-1" || app.Title != "Sales App" {
		t.Fatalf("unexpected application: %+v", app)
	}
}

func TestUpdateApplication(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		expected := applicationsPath + "('app-1')"
		if r.URL.Path != expected {
			t.Fatalf("expected path %q, got %q", expected, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := client.UpdateApplication(context.Background(), "app-1", Application{Title: "New Title"})
	if err != nil {
		t.Fatalf("UpdateApplication: %v", err)
	}
}

func TestDeleteApplication(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	if err := client.DeleteApplication(context.Background(), "app-1"); err != nil {
		t.Fatalf("DeleteApplication: %v", err)
	}
}

func TestDeleteApplication_AlreadyGoneIsNotFound(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	err := client.DeleteApplication(context.Background(), "app-1")
	var apiErr *apierror.Error
	if !errors.As(err, &apiErr) || !apiErr.IsNotFound() {
		t.Fatalf("expected a not-found *apierror.Error, got %v", err)
	}
}

func TestGetApplication_LoginPageIsAClearError(t *testing.T) {
	// What the Developer Hub's application router answers to a request
	// without a valid token: HTTP 200 with an HTML login redirect.
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><script>location="https://example.authentication.eu10.hana.ondemand.com/oauth/authorize"</script></head></html>`))
	})
	defer server.Close()

	_, err := client.GetApplication(context.Background(), "app-1")
	if !errors.Is(err, errLoginPage) {
		t.Fatalf("expected errLoginPage, got %v", err)
	}
	if _, err := client.GetServiceMetadata(context.Background()); !errors.Is(err, errLoginPage) {
		t.Fatalf("expected errLoginPage from $metadata too, got %v", err)
	}
}
