package provider

import "testing"

func TestSplitCompositeID(t *testing.T) {
	cases := []struct {
		id            string
		first, second string
		ok            bool
	}{
		{"app-1/sub-1", "app-1", "sub-1", true},
		{"app-1/sub-1/extra", "app-1", "sub-1/extra", true},
		{"app-1", "", "", false},
		{"/sub-1", "", "", false},
		{"app-1/", "", "", false},
		{"", "", "", false},
	}
	for _, tc := range cases {
		first, second, ok := splitCompositeID(tc.id)
		if ok != tc.ok || first != tc.first || second != tc.second {
			t.Errorf("splitCompositeID(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tc.id, first, second, ok, tc.first, tc.second, tc.ok)
		}
	}
}
