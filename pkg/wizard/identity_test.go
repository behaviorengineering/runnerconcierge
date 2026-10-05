package wizard

import (
	"testing"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

func TestRunnerIdentity_group(t *testing.T) {
	got, err := RunnerIdentity("behaviorengineering", "macos", "macstudio")
	if err != nil {
		t.Fatal(err)
	}
	if got != "behaviorengineering-macos-macstudio" {
		t.Fatalf("got %q", got)
	}
}

func TestRunnerIdentity_projectPath(t *testing.T) {
	got, err := RunnerIdentity("dss/conform", "ci", "macstudio")
	if err != nil {
		t.Fatal(err)
	}
	if got != "dss-conform-ci-macstudio" {
		t.Fatalf("got %q", got)
	}
}

func TestRunnerIdentity_missingTag(t *testing.T) {
	_, err := RunnerIdentity("acme", "", "host")
	if err == nil || errdefs.CodeOf(err) != errdefs.CodeInvalidScope {
		t.Fatalf("err %v", err)
	}
}
