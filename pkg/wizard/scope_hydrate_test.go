package wizard

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

type scopeHydrateExec struct{}

func (scopeHydrateExec) LookPath(name string) (string, error) { return name, nil }
func (scopeHydrateExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return scopeHydrateExec{}.Run(ctx, name, args...)
}
func (scopeHydrateExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "glab" && len(args) == 2 && args[0] == "api" && args[1] == "runners/12" {
		return json.Marshal(map[string]any{
			"id":          12,
			"runner_type": "group_type",
			"groups": []map[string]any{
				{"id": 3, "name": "acme", "web_url": "https://gitlab.com/groups/acme"},
			},
		})
	}
	return nil, nil
}

func TestHydrateRunnerScopeFromGitLab(t *testing.T) {
	cfg, _ := config.Load("")
	dir := t.TempDir()
	r := &Runner{
		cfg:   cfg,
		exec:  scopeHydrateExec{},
		store: &state.Store{Dir: dir},
		opts:  Options{TagList: []string{"lab"}},
	}
	cp := &state.Checkpoint{RunnerID: 12, Tags: []string{"lab"}, Executor: "docker"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.hydrateRunnerScopeFromGitLab(ctx, cp); err != nil {
		t.Fatal(err)
	}
	if r.opts.GroupPath != "acme" || r.opts.RunnerType != runnerTypeGroup {
		t.Fatalf("opts group=%q type=%q", r.opts.GroupPath, r.opts.RunnerType)
	}
	if cp.GroupPath != "acme" {
		t.Fatalf("cp group %q", cp.GroupPath)
	}
}
