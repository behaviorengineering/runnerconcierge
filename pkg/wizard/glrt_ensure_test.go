package wizard

import (
	"context"
	"testing"
	"time"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

type resetGLRTExec struct{}

func (resetGLRTExec) LookPath(name string) (string, error) { return name, nil }
func (resetGLRTExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return resetGLRTExec{}.Run(ctx, name, args...)
}
func (resetGLRTExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "glab" && len(args) >= 4 && args[3] == "runners/12/reset_authentication_token" {
		return []byte(`{"token":"glrt-resume"}`), nil
	}
	return nil, nil
}

func TestResetRunnerGLRTIfNeeded_storesFromAPI(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	cp := &state.Checkpoint{RunnerID: 12}
	r := &Runner{
		exec: resetGLRTExec{},
		opts: Options{Keyring: mem, TagList: []string{"lab"}},
		cfg:  &config.UserConfig{GitLabURL: "https://gitlab.com"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.resetRunnerGLRTIfNeeded(ctx, cp); err != nil {
		t.Fatal(err)
	}
	if r.opts.RunnerToken != "glrt-resume" {
		t.Fatalf("token %q", r.opts.RunnerToken)
	}
	got, err := config.LoadRunnerTokenByTag("lab", mem)
	if err != nil || got != "glrt-resume" {
		t.Fatalf("by tag %q err %v", got, err)
	}
	gotID, err := config.LoadRunnerToken(12, mem)
	if err != nil || gotID != "glrt-resume" {
		t.Fatalf("by id %q err %v", gotID, err)
	}
}

func TestResetRunnerGLRTIfNeeded_skipsWhenKeyringHit(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	if err := config.StoreRunnerTokenByTag("lab", "glrt-saved", mem); err != nil {
		t.Fatal(err)
	}
	cp := &state.Checkpoint{RunnerID: 3}
	r := &Runner{
		exec: resetGLRTExec{},
		opts: Options{Keyring: mem, TagList: []string{"lab"}},
		cfg:  &config.UserConfig{GitLabURL: "https://gitlab.com"},
	}
	if err := r.loadGLRTIntoOpts(cp.RunnerID, cp); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.resetRunnerGLRTIfNeeded(ctx, cp); err != nil {
		t.Fatal(err)
	}
	if r.opts.RunnerToken != "glrt-saved" {
		t.Fatalf("token %q", r.opts.RunnerToken)
	}
}

func TestPromptCanonicalTag_skipsWhenHydrated(t *testing.T) {
	pr := &recordingPrompter{}
	cp := &state.Checkpoint{Tags: []string{"from-cp"}}
	r := &Runner{opts: Options{TagList: []string{"hydrated"}}}
	if err := r.promptCanonicalTag(context.Background(), pr, cp); err != nil {
		t.Fatal(err)
	}
	if len(r.opts.TagList) != 1 || r.opts.TagList[0] != "hydrated" {
		t.Fatalf("tags %v", r.opts.TagList)
	}
}
