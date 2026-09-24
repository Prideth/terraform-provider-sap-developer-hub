package apierror

import "testing"

func TestError_MessageFormatting(t *testing.T) {
	withCode := &Error{StatusCode: 409, Code: "CONFLICT", Message: "already exists"}
	if withCode.Error() != `SAP Developer Hub API returned HTTP 409, code "CONFLICT": already exists` {
		t.Fatalf("unexpected message: %q", withCode.Error())
	}

	withoutCode := &Error{StatusCode: 500, Message: "internal error"}
	if withoutCode.Error() != "SAP Developer Hub API returned HTTP 500: internal error" {
		t.Fatalf("unexpected message: %q", withoutCode.Error())
	}
}

func TestError_StatusPredicates(t *testing.T) {
	cases := []struct {
		status                           int
		notFound, conflict, unauthorized bool
	}{
		{404, true, false, false},
		{409, false, true, false},
		{401, false, false, true},
		{200, false, false, false},
	}
	for _, tc := range cases {
		e := &Error{StatusCode: tc.status}
		if e.IsNotFound() != tc.notFound {
			t.Errorf("status %d: IsNotFound() = %v, want %v", tc.status, e.IsNotFound(), tc.notFound)
		}
		if e.IsConflict() != tc.conflict {
			t.Errorf("status %d: IsConflict() = %v, want %v", tc.status, e.IsConflict(), tc.conflict)
		}
		if e.IsUnauthorized() != tc.unauthorized {
			t.Errorf("status %d: IsUnauthorized() = %v, want %v", tc.status, e.IsUnauthorized(), tc.unauthorized)
		}
	}
}

func TestError_NilReceiverIsSafe(t *testing.T) {
	var e *Error
	if e.IsNotFound() || e.IsConflict() || e.IsUnauthorized() {
		t.Fatal("expected all predicates to be false on a nil *Error")
	}
}
