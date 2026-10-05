package wizard

import (
	"testing"

	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

func TestRunnerName_usesIdentity(t *testing.T) {
	cp := &state.Checkpoint{GroupPath: "behaviorengineering", Tags: []string{"macos"}}
	r := &Runner{opts: Options{GroupPath: "behaviorengineering", TagList: []string{"macos"}}}
	name := r.runnerName(cp)
	want, err := RunnerIdentity("behaviorengineering", "macos", hostname())
	if err != nil {
		t.Fatal(err)
	}
	if name != want {
		t.Fatalf("name %q want %q", name, want)
	}
}
