package developerhub

import "testing"

func TestParseError_ODataEnvelope(t *testing.T) {
	body := []byte(`{"error":{"code":"CONFLICT","message":{"lang":"en","value":"already exists"}}}`)
	err := parseError(409, body)
	if err.Code != "CONFLICT" || err.Message != "already exists" || err.StatusCode != 409 {
		t.Fatalf("unexpected error: %+v", err)
	}
	if !err.IsConflict() {
		t.Fatal("expected IsConflict() to be true")
	}
}

func TestParseError_FlatMessage(t *testing.T) {
	body := []byte(`{"message":"invalid request"}`)
	err := parseError(400, body)
	if err.Message != "invalid request" {
		t.Fatalf("unexpected error: %+v", err)
	}
}

func TestParseError_FlatErrorString(t *testing.T) {
	body := []byte(`{"error":"bad credentials"}`)
	err := parseError(401, body)
	if err.Message != "bad credentials" {
		t.Fatalf("unexpected error: %+v", err)
	}
	if !err.IsUnauthorized() {
		t.Fatal("expected IsUnauthorized() to be true")
	}
}

func TestParseError_UnparseableBodyFallsBackToRawText(t *testing.T) {
	body := []byte(`<html>not json</html>`)
	err := parseError(500, body)
	if err.Message == "" {
		t.Fatal("expected a non-empty fallback message")
	}
	if err.StatusCode != 500 {
		t.Fatalf("expected status 500, got %d", err.StatusCode)
	}
}

func TestParseError_EmptyBody(t *testing.T) {
	err := parseError(503, nil)
	if err.Message != "no error details returned" {
		t.Fatalf("unexpected message: %q", err.Message)
	}
}
