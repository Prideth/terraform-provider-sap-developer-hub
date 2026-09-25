package developerhub

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateSubscription(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != subscriptionsPath {
			t.Fatalf("expected path %q, got %q", subscriptionsPath, r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var body subscriptionWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		if body.ID != placeholderID {
			t.Fatalf("expected placeholder id %q, got %q", placeholderID, body.ID)
		}
		if len(body.APIProduct) != 1 || body.APIProduct[0].Metadata.URI != "APIMgmt.APIProducts('Sales_API')" {
			t.Fatalf("unexpected product reference: %+v", body.APIProduct)
		}
		if len(body.Application) != 1 || body.Application[0].Metadata.URI != "APIMgmt.Applications('app-1')" {
			t.Fatalf("unexpected application reference: %+v", body.Application)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sub-1","app_id":"app-1","product_id":"Sales_API","isSubscribed":true,"status":"active"}`))
	})
	defer server.Close()

	sub, err := client.CreateSubscription(context.Background(), "app-1", "Sales_API")
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}
	if sub.ID != "sub-1" || sub.AppID != "app-1" || sub.ProductID != "Sales_API" || sub.Status != "active" {
		t.Fatalf("unexpected subscription: %+v", sub)
	}
}

func TestGetSubscription(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		expected := subscriptionsPath + "('sub-1')"
		if r.URL.Path != expected {
			t.Fatalf("expected path %q, got %q", expected, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sub-1","app_id":"app-1","product_id":"Sales_API","developer_id":"dev-1","isSubscribed":true,"status":"pending approval"}`))
	})
	defer server.Close()

	sub, err := client.GetSubscription(context.Background(), "sub-1")
	if err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}
	if sub.AppID != "app-1" || sub.ProductID != "Sales_API" || sub.Status != "pending approval" {
		t.Fatalf("unexpected subscription: %+v", sub)
	}
}

func TestUpdateSubscriptionProduct(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		expected := subscriptionsPath + "('sub-1')"
		if r.URL.Path != expected {
			t.Fatalf("expected path %q, got %q", expected, r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		var body subscriptionWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		if body.ProductID != "Order_API" {
			t.Fatalf("expected product_id %q, got %q", "Order_API", body.ProductID)
		}
		if len(body.APIProduct) != 1 || body.APIProduct[0].Metadata.URI != "APIMgmt.APIProducts('Order_API')" {
			t.Fatalf("unexpected product reference: %+v", body.APIProduct)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	if err := client.UpdateSubscriptionProduct(context.Background(), "sub-1", "Order_API"); err != nil {
		t.Fatalf("UpdateSubscriptionProduct: %v", err)
	}
}

func TestDeleteSubscription(t *testing.T) {
	var called bool
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		expected := subscriptionsPath + "('sub-1')"
		if r.URL.Path != expected {
			t.Fatalf("expected path %q, got %q", expected, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	if err := client.DeleteSubscription(context.Background(), "sub-1"); err != nil {
		t.Fatalf("DeleteSubscription: %v", err)
	}
	if !called {
		t.Fatal("expected the server to be called")
	}
}
