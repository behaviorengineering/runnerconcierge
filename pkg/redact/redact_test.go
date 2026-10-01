package redact

import "testing"

func TestString(t *testing.T) {
	in := "token glrt-abc123xyz and glpat-foo"
	got := String(in)
	if got == in {
		t.Fatalf("expected redaction, got %q", got)
	}
	if contains(got, "glrt-abc") || contains(got, "glpat-foo") {
		t.Fatalf("secrets leaked: %q", got)
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && search(s, sub)
}

func search(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
