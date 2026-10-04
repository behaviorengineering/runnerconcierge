package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunBareInvokeAgentGuide(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), nil, &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(out.String(), "setup complete") {
		t.Fatal("bare invoke should not run setup")
	}
	if !strings.Contains(out.String(), "Documentation for agents") {
		t.Fatalf("expected agent guide: %q", out.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) < 10 {
		t.Fatalf("expected multi-line guide, got %d lines: %q", len(lines), out.String())
	}
	if strings.Contains(out.String(), "Bare invoke runs the interactive") {
		t.Fatal("stale bare-invoke wizard wording")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"not-a-command"}, &out, &errOut)
	if code != ExitUsage {
		t.Fatalf("exit %d", code)
	}
}

func TestRunnersNeedForge(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"runners"}, &out, &errOut)
	if code == ExitOK {
		t.Fatal("expected failure without forge")
	}
	if !strings.Contains(errOut.String(), "choose a forge") {
		t.Fatalf("stderr: %q", errOut.String())
	}
}

func TestRunnersGitHubUnsupported(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"runners", "github"}, &out, &errOut)
	if code == ExitOK {
		t.Fatal("expected unsupported_forge")
	}
	if !strings.Contains(errOut.String(), "unsupported_forge") {
		t.Fatalf("stderr: %q", errOut.String())
	}
}

func TestRunVersion(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"version"}, &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "runnerconcierge") {
		t.Fatalf("version output: %q", out.String())
	}
}
