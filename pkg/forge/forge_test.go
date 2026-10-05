package forge

import (
	"strings"
	"testing"
)

func TestNeedSubcommand_message(t *testing.T) {
	err := NeedSubcommand(VerbRunners, GitLab)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "list") || !strings.Contains(msg, "setup") {
		t.Fatalf("message: %q", msg)
	}
	if !strings.Contains(msg, "runners gitlab") {
		t.Fatalf("message: %q", msg)
	}
}
