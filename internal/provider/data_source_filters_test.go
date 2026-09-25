package provider

import (
	"testing"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/developerhub"
)

func TestFilterApplications(t *testing.T) {
	apps := []developerhub.Application{
		{ID: "a1", Title: "One", DeveloperID: "dev-1"},
		{ID: "a2", Title: "Two", DeveloperID: "dev-2"},
	}
	if got := filterApplications(apps, ""); len(got) != 2 {
		t.Fatalf("expected no filtering without developer_id, got %d", len(got))
	}
	got := filterApplications(apps, "dev-2")
	if len(got) != 1 || got[0].ID.ValueString() != "a2" {
		t.Fatalf("expected only a2, got %+v", got)
	}
}

func TestFilterSubscriptions(t *testing.T) {
	subs := []developerhub.Subscription{
		{ID: "s1", AppID: "a1", ProductID: "P1", Status: "approved"},
		{ID: "s2", AppID: "a1", ProductID: "P2"},
		{ID: "s3", AppID: "a2", ProductID: "P1", IsSubscribed: true},
	}
	cases := []struct {
		app, product string
		want         []string
	}{
		{"", "", []string{"s1", "s2", "s3"}},
		{"a1", "", []string{"s1", "s2"}},
		{"", "P1", []string{"s1", "s3"}},
		{"a2", "P1", []string{"s3"}},
		{"a2", "P2", nil},
	}
	for _, tc := range cases {
		got := filterSubscriptions(subs, tc.app, tc.product)
		if len(got) != len(tc.want) {
			t.Errorf("app=%q product=%q: got %d results, want %d", tc.app, tc.product, len(got), len(tc.want))
			continue
		}
		for i, id := range tc.want {
			if got[i].ID.ValueString() != id {
				t.Errorf("app=%q product=%q: result %d = %s, want %s", tc.app, tc.product, i, got[i].ID.ValueString(), id)
			}
		}
	}
	if got := filterSubscriptions(subs, "a2", ""); !got[0].IsSubscribed.ValueBool() {
		t.Error("expected is_subscribed to be carried over")
	}
}
