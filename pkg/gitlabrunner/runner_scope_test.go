package gitlabrunner

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type runnerScopeExec struct{}

func (runnerScopeExec) LookPath(name string) (string, error) { return name, nil }
func (runnerScopeExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return runnerScopeExec{}.Run(ctx, name, args...)
}
func (runnerScopeExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "glab" && len(args) == 2 && args[0] == "api" && args[1] == "runners/57039107" {
		return json.Marshal(map[string]any{
			"id":          57039107,
			"runner_type": "group_type",
			"tag_list":    []string{"macos"},
			"groups": []map[string]any{
				{
					"id":      140366149,
					"name":    "behaviorengineering",
					"web_url": "https://gitlab.com/groups/behaviorengineering",
				},
			},
		})
	}
	return nil, nil
}

func TestResolveRunnerScope_group(t *testing.T) {
	client := NewClient("https://gitlab.com", runnerScopeExec{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	runnerType, path, err := client.ResolveRunnerScope(ctx, 57039107)
	if err != nil {
		t.Fatal(err)
	}
	if runnerType != "group_type" || path != "behaviorengineering" {
		t.Fatalf("type=%q path=%q", runnerType, path)
	}
}
