package errdefs

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestErrorOmitsCauseString(t *testing.T) {
	cause := fmt.Errorf("glab api --method POST user/runners: HTTP 400")
	err := New("gitlabrunner.CreateRunner", CodeCreateFailed, "GitLab rejected the request", cause)
	got := err.Error()
	if strings.Contains(got, "glab api --method") {
		t.Fatalf("Error() leaked cause argv: %q", got)
	}
	if !strings.Contains(got, "code=create_failed") {
		t.Fatalf("expected code in Error(): %q", got)
	}
	if !errors.Is(err, cause) {
		t.Fatal("Unwrap should reach cause")
	}
}

func TestFormatCLI(t *testing.T) {
	err := New("setup", CodeInvalidScope, "GitLab project path is required (--repo)", nil)
	if FormatCLI(err) != err.Error() {
		t.Fatalf("FormatCLI domain: %q", FormatCLI(err))
	}
}
