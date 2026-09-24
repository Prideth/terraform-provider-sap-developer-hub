package developerhub

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateSubscription(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		expected := applicationsPath + "('app-1')/ToSubscriptions"
		if r.URL.Path != expected {
			t.Fatalf("expected path %q, got %q", expected, r.URL.Path)
		}
		var body subscriptionCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		if len(body.APIProduct) != 1 || body.APIProduct[0].Metadata.URI != "APIMgmt.APIProducts('Sales_API')" {
			t.Fatalf("unexpected product reference: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sub-1"}`))
	})
	defer server.Close()

	sub, err := client.CreateSubscription(context.Background(), "app-1", "Sales_API")
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}
	if sub.ID != "sub-1" || sub.ProductName != "Sales_API" {
		t.Fatalf("unexpected subscription: %+v", sub)
	}
}

func TestGetSubscription_ExtractsProductNameFromReference(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		expected := applicationsPath + "('app-1')/ToSubscriptions('sub-1')"
		if r.URL.Path != expected {
			t.Fatalf("expected path %q, got %q", expected, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sub-1","ToAPIProduct":[{"__metadata":{"uri":"APIMgmt.APIProducts('Sales_API')"}}]}`))
	})
	defer server.Close()

	sub, err := client.GetSubscription(context.Background(), "app-1", "sub-1")
	if err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}
	if sub.ProductName != "Sales_API" {
		t.Fatalf("expected product name %q, got %q", "Sales_API", sub.ProductName)
	}
}

func TestDeleteSubscription(t *testing.T) {
	var called bool
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	if err := client.DeleteSubscription(context.Background(), "app-1", "sub-1"); err != nil {
		t.Fatalf("DeleteSubscription: %v", err)
	}
	if !called {
		t.Fatal("expected the server to be called")
	}
}

func TestProductNameFromReferenceURI(t *testing.T) {
	cases := map[string]string{
		"APIMgmt.APIProducts('Sales_API')":    "Sales_API",
		"APIMgmt.APIProducts('O''Brien_API')": "O'Brien_API",
		"not-a-reference":                     "",
	}
	for uri, want := range cases {
		if got := productNameFromReferenceURI(uri); got != want {
			t.Errorf("productNameFromReferenceURI(%q) = %q, want %q", uri, got, want)
		}
	}
}
