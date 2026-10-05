package wizard

import (
	"os"
	"strings"
	"testing"

	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

func TestBeginFromStoredCheckpoint_archivesDone(t *testing.T) {
	dir := t.TempDir()
	store := &state.Store{Dir: dir}
	cp := &state.Checkpoint{
		Version:  1,
		Stage:    state.CheckpointStageDone,
		RunnerID: 57039107,
		Tags:     []string{"macos"},
	}
	if err := store.Save(cp); err != nil {
		t.Fatal(err)
	}
	var lines []string
	r := &Runner{store: store, out: func(s string) { lines = append(lines, s) }}
	got, found := r.beginFromStoredCheckpoint(cp, true)
	if found {
		t.Fatal("done checkpoint must not resume")
	}
	if got == nil || got.RunnerID != 57039107 {
		t.Fatalf("checkpoint %+v", got)
	}
	if _, err := os.Stat(store.Path()); !os.IsNotExist(err) {
		t.Fatalf("state.json still present: %v", err)
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "starting a new runner setup") {
		t.Fatalf("out %v", lines)
	}
	if !strings.Contains(lines[0], "runner_id=57039107") || !strings.Contains(lines[0], "macos") {
		t.Fatalf("summary %q", lines[0])
	}
}

func TestBeginFromStoredCheckpoint_keepsIncomplete(t *testing.T) {
	r := &Runner{out: func(string) {}}
	cp := &state.Checkpoint{Stage: "identity", RunnerID: 12}
	got, found := r.beginFromStoredCheckpoint(cp, true)
	if !found || got != cp {
		t.Fatalf("found=%v cp=%v", found, got)
	}
}
