package wizard

import (
	"context"
	"fmt"
	"testing"

	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

type selectFailPrompter struct {
	recordingPrompter
}

func (s *selectFailPrompter) Select(ctx context.Context, title string, options []string) (int, error) {
	return 0, fmt.Errorf("select called: %s", title)
}

func TestPromptSetup_skipsProjectWhenHydrated(t *testing.T) {
	cfg, _ := config.Load("")
	cp := &state.Checkpoint{RepoPath: "acme/widget"}
	r := &Runner{
		cfg:   cfg,
		opts:  Options{ProjectPath: "acme/widget", RunnerType: runnerTypeProject, TagList: []string{"lab"}, Executor: "shell"},
		store: &state.Store{Dir: t.TempDir()},
	}
	pr := &selectFailPrompter{}
	if err := r.promptSetupOptions(context.Background(), pr, cp); err != nil {
		t.Fatal(err)
	}
	if cp.RepoPath != "acme/widget" {
		t.Fatalf("repo %q", cp.RepoPath)
	}
}
