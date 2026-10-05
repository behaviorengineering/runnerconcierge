package wizard

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

type groupListExec struct{}

func (groupListExec) LookPath(name string) (string, error) { return name, nil }
func (groupListExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return groupListExec{}.Run(ctx, name, args...)
}
func (groupListExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "glab" && len(args) >= 2 && args[0] == "api" && strings.HasPrefix(args[1], "groups?") {
		if !strings.Contains(args[1], "min_access_level=40") {
			return nil, errors.New("missing min_access_level")
		}
		body, _ := json.Marshal([]map[string]any{{"id": 1, "full_path": "behaviorengineering", "name": "BE"}})
		return body, nil
	}
	return nil, nil
}

func TestPromptSetup_promptsGroupWhenTokenWithoutScope(t *testing.T) {
	cfg, _ := config.Load("")
	cp := &state.Checkpoint{Tags: []string{"macos-dss"}}
	r := &Runner{
		cfg:   cfg,
		exec:  groupListExec{},
		opts:  Options{RunnerToken: "glrt-old", TagList: []string{"macos-dss"}, RunnerType: runnerTypeGroup},
		store: &state.Store{Dir: t.TempDir()},
	}
	pr := &selectFailPrompter{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := r.promptSetupOptions(ctx, pr, cp)
	if err == nil || !strings.Contains(err.Error(), "select called") {
		t.Fatalf("expected group prompt err, got %v", err)
	}
}

type inputFailPrompter struct {
	recordingPrompter
}

func (i *inputFailPrompter) Input(ctx context.Context, title, placeholder, value string) (string, error) {
	if strings.Contains(title, "tag") {
		return "", errors.New("unexpected tag prompt: " + title)
	}
	return i.recordingPrompter.Input(ctx, title, placeholder, value)
}

func TestPromptSetup_skipsSecondNamePrompt(t *testing.T) {
	cfg, _ := config.Load("")
	cp := &state.Checkpoint{Tags: []string{"macos-dss"}}
	r := &Runner{
		cfg:   cfg,
		exec:  groupListExec{},
		opts:  Options{TagList: []string{"macos-dss"}, GroupPath: "behaviorengineering", RunnerType: runnerTypeGroup, Executor: "docker"},
		store: &state.Store{Dir: t.TempDir()},
	}
	pr := &inputFailPrompter{}
	if err := r.promptSetupOptions(context.Background(), pr, cp); err != nil {
		t.Fatal(err)
	}
}
