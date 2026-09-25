package developerhub

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestListApplications_FollowsRelativeNextLink(t *testing.T) {
	var calls int
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("$skiptoken") == "" {
			_, _ = w.Write([]byte(`{"d":{"results":[{"id":"app-1","title":"One"}],"__next":"APIMgmt.Applications?$skiptoken=2"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"d":{"results":[{"id":"app-2","title":"Two"}]}}`))
	})
	defer server.Close()

	apps, err := client.ListApplications(context.Background())
	if err != nil {
		t.Fatalf("ListApplications: %v", err)
	}
	if len(apps) != 2 || apps[0].ID != "app-1" || apps[1].ID != "app-2" || calls != 2 {
		t.Fatalf("expected both pages in order across 2 calls, got %+v after %d calls", apps, calls)
	}
}

func TestListSubscriptions_FollowsAbsoluteNextLinkOnSameHost(t *testing.T) {
	var serverURL string
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != subscriptionsPath {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("$skiptoken") == "" {
			_, _ = w.Write([]byte(`{"d":{"results":[{"id":"sub-1","app_id":"app-1","product_id":"P1"}],"__next":"` +
				serverURL + subscriptionsPath + `?$skiptoken=2"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"d":{"results":[{"id":"sub-2","app_id":"app-2","product_id":"P2"}]}}`))
	})
	defer server.Close()
	serverURL = server.URL

	subs, err := client.ListSubscriptions(context.Background())
	if err != nil {
		t.Fatalf("ListSubscriptions: %v", err)
	}
	if len(subs) != 2 || subs[1].ProductID != "P2" {
		t.Fatalf("unexpected subscriptions: %+v", subs)
	}
}

func TestListAll_RefusesNextLinkToAnotherHost(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"d":{"results":[],"__next":"https://attacker.example/steal"}}`))
	})
	defer server.Close()

	_, err := client.ListApplications(context.Background())
	if err == nil || !strings.Contains(err.Error(), "different host") {
		t.Fatalf("expected a refusal to follow a cross-host paging link, got %v", err)
	}
}

func TestListAll_StopsAfterMaxPages(t *testing.T) {
	var calls int
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"d":{"results":[],"__next":"APIMgmt.Applications?$skiptoken=again"}}`))
	})
	defer server.Close()

	_, err := client.ListApplications(context.Background())
	if err == nil || !strings.Contains(err.Error(), "more than") {
		t.Fatalf("expected an error after hitting the page limit, got %v", err)
	}
	if calls != maxPages {
		t.Fatalf("expected exactly %d requests, got %d", maxPages, calls)
	}
}

func TestListApplications_PlainArray(t *testing.T) {
	client, server := testDevPortalClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"app-1","title":"One","description":"d","callbackurl":"https://cb"}]`))
	})
	defer server.Close()

	apps, err := client.ListApplications(context.Background())
	if err != nil {
		t.Fatalf("ListApplications: %v", err)
	}
	if len(apps) != 1 || apps[0].CallbackURL != "https://cb" {
		t.Fatalf("unexpected applications: %+v", apps)
	}
}
