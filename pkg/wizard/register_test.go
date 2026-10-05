package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

type registerRetryExec struct {
	registers int
}

func (e *registerRetryExec) LookPath(name string) (string, error) { return name, nil }
func (e *registerRetryExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return e.Run(ctx, name, args...)
}
func (e *registerRetryExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if len(args) > 0 && args[0] == "register" {
		e.registers++
		if e.registers == 1 {
			return nil, errors.New("Verifying runner... is not valid")
		}
		return []byte("ok"), nil
	}
	if name == "glab" && len(args) >= 4 && args[3] == "runners/12/reset_authentication_token" {
		return []byte(`{"token":"glrt-resume"}`), nil
	}
	return nil, nil
}

func TestInvalidRegisterToken(t *testing.T) {
	if invalidRegisterToken(nil) {
		t.Fatal("nil")
	}
	if invalidRegisterToken(errors.New("boom")) {
		t.Fatal("unrelated")
	}
	inner := errdefs.New("Register", errdefs.CodeRegisterFailed, "register failed: Verifying runner... is not valid", nil)
	outer := errdefs.New("setup", errdefs.CodeRegisterFailed, "could not build register argv", inner)
	if !invalidRegisterToken(inner) || !invalidRegisterToken(outer) {
		t.Fatal("expected invalid token detection")
	}
}

func TestRegisterWithRetry_resetsInvalidToken(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	exec := &registerRetryExec{}
	cp := &state.Checkpoint{RunnerID: 12, GroupPath: "acme", Tags: []string{"lab"}}
	r := &Runner{
		exec: exec,
		opts: Options{Keyring: mem, GroupPath: "acme", TagList: []string{"lab"}, RunnerToken: "glrt-stale", Executor: "shell"},
		cfg:  &config.UserConfig{GitLabURL: "https://gitlab.com", DefaultExecutor: "shell"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.registerWithRetry(ctx, nil, "gitlab-runner", cp, "glrt-stale", "", ""); err != nil {
		t.Fatal(err)
	}
	if exec.registers != 2 {
		t.Fatalf("registers %d", exec.registers)
	}
	if r.opts.RunnerToken != "glrt-resume" {
		t.Fatalf("token %q", r.opts.RunnerToken)
	}
}

func TestRegisterOnce_wrapsArgvError(t *testing.T) {
	r := &Runner{cfg: &config.UserConfig{}}
	err := r.registerOnce(context.Background(), "gitlab-runner", &state.Checkpoint{}, "", "", "")
	if err == nil || !strings.Contains(err.Error(), "register") {
		t.Fatalf("err %v", err)
	}
}
