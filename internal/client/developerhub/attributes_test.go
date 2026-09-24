package developerhub

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestListApplicationAttributes_PlainArray(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		expected := applicationsPath + "('app-1')/ToAttributes"
		if r.URL.Path != expected {
			t.Fatalf("expected path %q, got %q", expected, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"env","value":"prod","entityType":"Applications","entityId":"app-1"}]`))
	})
	defer server.Close()

	attrs, err := client.ListApplicationAttributes(context.Background(), "app-1")
	if err != nil {
		t.Fatalf("ListApplicationAttributes: %v", err)
	}
	if len(attrs) != 1 || attrs[0].Name != "env" || attrs[0].Value != "prod" {
		t.Fatalf("unexpected attributes: %+v", attrs)
	}
}

func TestListApplicationAttributes_ResultsEnvelope(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"name":"env","value":"prod","entityType":"Applications","entityId":"app-1"}]}`))
	})
	defer server.Close()

	attrs, err := client.ListApplicationAttributes(context.Background(), "app-1")
	if err != nil {
		t.Fatalf("ListApplicationAttributes: %v", err)
	}
	if len(attrs) != 1 || attrs[0].Name != "env" {
		t.Fatalf("unexpected attributes: %+v", attrs)
	}
}

func TestCreateApplicationAttribute(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		expected := applicationsPath + "('app-1')/ToAttributes"
		if r.URL.Path != expected {
			t.Fatalf("expected path %q, got %q", expected, r.URL.Path)
		}
		var body Attribute
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		if body.EntityType != "Applications" || body.EntityID != "app-1" {
			t.Fatalf("unexpected entity reference: %+v", body)
		}
		w.WriteHeader(http.StatusCreated)
	})
	defer server.Close()

	if err := client.CreateApplicationAttribute(context.Background(), "app-1", "env", "prod"); err != nil {
		t.Fatalf("CreateApplicationAttribute: %v", err)
	}
}

func TestUpdateAndDeleteApplicationAttribute(t *testing.T) {
	expectedPath := attributesPath + "(name='env',entityId='app-1',entityType='Applications')"
	var lastMethod string
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expectedPath {
			t.Fatalf("expected path %q, got %q", expectedPath, r.URL.Path)
		}
		lastMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	if err := client.UpdateApplicationAttribute(context.Background(), "app-1", "env", "staging"); err != nil {
		t.Fatalf("UpdateApplicationAttribute: %v", err)
	}
	if lastMethod != http.MethodPut {
		t.Fatalf("expected PUT, got %s", lastMethod)
	}

	if err := client.DeleteApplicationAttribute(context.Background(), "app-1", "env"); err != nil {
		t.Fatalf("DeleteApplicationAttribute: %v", err)
	}
	if lastMethod != http.MethodDelete {
		t.Fatalf("expected DELETE, got %s", lastMethod)
	}
}

func TestOdataKey_EscapesQuotes(t *testing.T) {
	if got := odataKey("o'brien"); got != "'o''brien'" {
		t.Fatalf("unexpected escaped key: %q", got)
	}
}
