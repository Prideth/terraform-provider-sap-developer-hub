package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/developerhub"
	sapthttp "github.com/Prideth/terraform-provider-sap-developer-hub/internal/client/http"
)

// TestReconcileAttributes_IssuesMinimalCalls verifies that updating an
// application's attributes issues exactly one call per changed attribute -
// a create for a new name, an update for a changed value, a delete for a
// removed name - and no call at all for an attribute that did not change,
// since the Developer Hub API has no bulk "replace all attributes"
// endpoint (DESIGN.md §6).
func TestReconcileAttributes_IssuesMinimalCalls(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	httpClient := sapthttp.New(sapthttp.Config{Transport: http.DefaultClient, UserAgent: "test"})
	res := &applicationResource{client: developerhub.New(httpClient, server.URL)}

	previous := []applicationAttributeModel{
		{Name: types.StringValue("unchanged"), Value: types.StringValue("same")},
		{Name: types.StringValue("changed"), Value: types.StringValue("old")},
		{Name: types.StringValue("removed"), Value: types.StringValue("gone-soon")},
	}
	desired := []applicationAttributeModel{
		{Name: types.StringValue("unchanged"), Value: types.StringValue("same")},
		{Name: types.StringValue("changed"), Value: types.StringValue("new")},
		{Name: types.StringValue("added"), Value: types.StringValue("brand-new")},
	}

	if err := res.reconcileAttributes(context.Background(), "app-1", previous, desired); err != nil {
		t.Fatalf("reconcileAttributes: %v", err)
	}

	if len(calls) != 3 {
		t.Fatalf("expected exactly 3 calls (create+update+delete), got %d: %v", len(calls), calls)
	}

	const (
		applicationsPath = "/odata/1.0/data.svc/APIMgmt.Applications"
		attributesPath   = "/odata/1.0/data.svc/APIMgmt.Attributes"
	)

	var sawCreateAdded, sawUpdateChanged, sawDeleteRemoved bool
	for _, c := range calls {
		switch {
		case c == "POST "+applicationsPath+"('app-1')/ToAttributes":
			sawCreateAdded = true
		case c == "PUT "+attributesPath+"(name='changed',entityId='app-1',entityType='Applications')":
			sawUpdateChanged = true
		case c == "DELETE "+attributesPath+"(name='removed',entityId='app-1',entityType='Applications')":
			sawDeleteRemoved = true
		}
	}
	if !sawCreateAdded || !sawUpdateChanged || !sawDeleteRemoved {
		t.Fatalf("missing an expected call, got: %v", calls)
	}
}

func TestReconcileAttributes_NoChangesIssuesNoCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected call: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	httpClient := sapthttp.New(sapthttp.Config{Transport: http.DefaultClient, UserAgent: "test"})
	res := &applicationResource{client: developerhub.New(httpClient, server.URL)}

	same := []applicationAttributeModel{
		{Name: types.StringValue("env"), Value: types.StringValue("prod")},
	}

	if err := res.reconcileAttributes(context.Background(), "app-1", same, same); err != nil {
		t.Fatalf("reconcileAttributes: %v", err)
	}
}
