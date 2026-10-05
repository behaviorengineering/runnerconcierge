package wizard

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

type recordingPrompter struct {
	passwords []string
	passwordN int
	confirm   bool
}

func (r *recordingPrompter) Confirm(ctx context.Context, title string) (bool, error) {
	if title == glabLoginConfirmTitle {
		return r.confirm, nil
	}
	return true, nil
}

func (r *recordingPrompter) Input(ctx context.Context, title, placeholder, value string) (string, error) {
	return value, nil
}

func (r *recordingPrompter) Password(ctx context.Context, title string) (string, error) {
	if r.passwordN < len(r.passwords) {
		val := r.passwords[r.passwordN]
		r.passwordN++
		return val, nil
	}
	return "", nil
}

func (r *recordingPrompter) Select(ctx context.Context, title string, options []string) (int, error) {
	return 0, nil
}

type loginExec struct {
	loginCalls int
}

func (l *loginExec) LookPath(name string) (string, error) { return name, nil }
func (l *loginExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return l.Run(ctx, name, args...)
}
func (l *loginExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "glab" && len(args) >= 1 && args[0] == "auth" {
		l.loginCalls++
		return nil, nil
	}
	if name == "glab" && len(args) == 2 && args[0] == "api" && args[1] == "user" {
		if l.loginCalls == 0 {
			return nil, errLoginExec("not logged in")
		}
		return json.Marshal(map[string]any{"id": 9, "username": "mindhoc"})
	}
	return nil, nil
}

type errLoginExec string

func (e errLoginExec) Error() string { return string(e) }

func testRunner(exec *loginExec) *Runner {
	return &Runner{
		exec: exec,
		opts: Options{},
		cfg:  &config.UserConfig{GitLabURL: "https://gitlab.com"},
	}
}

func TestHydrateRunnerToken_skipsTagWithoutRunnerID(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	if err := config.StoreRunnerTokenByTag("lab", "glrt-saved", mem); err != nil {
		t.Fatal(err)
	}
	r := &Runner{opts: Options{Keyring: mem, TagList: []string{"lab"}, GroupPath: "acme"}}
	if err := r.hydrateRunnerToken(&state.Checkpoint{GroupPath: "acme", Tags: []string{"lab"}}); err != nil {
		t.Fatal(err)
	}
	if r.opts.RunnerToken != "" {
		t.Fatalf("token %q", r.opts.RunnerToken)
	}
}

func TestHydrateRunnerToken_loadsIdentityWithRunnerID(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	cp := &state.Checkpoint{RunnerID: 55, GroupPath: "acme", Tags: []string{"lab"}}
	opts := Options{Keyring: mem, TagList: []string{"lab"}, GroupPath: "acme"}
	identity, err := runnerIdentityFor(opts, cp)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.StoreRunnerTokenByIdentity(identity, "glrt-id", mem); err != nil {
		t.Fatal(err)
	}
	r := &Runner{opts: opts}
	if err := r.hydrateRunnerToken(cp); err != nil {
		t.Fatal(err)
	}
	if r.opts.RunnerToken != "glrt-id" {
		t.Fatalf("token %q", r.opts.RunnerToken)
	}
}

func TestEnsureLoggedIn_runsGlabAuthLogin(t *testing.T) {
	exec := &loginExec{}
	r := testRunner(exec)
	pr := &recordingPrompter{confirm: true}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	user, err := r.ensureLoggedIn(ctx, pr)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != 9 || exec.loginCalls != 1 {
		t.Fatalf("user %+v loginCalls %d", user, exec.loginCalls)
	}
	if pr.passwordN != 0 {
		t.Fatalf("password calls %d", pr.passwordN)
	}
}

func TestEnsureLoggedIn_declineConfirmFails(t *testing.T) {
	r := testRunner(&loginExec{})
	pr := &recordingPrompter{confirm: false}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.ensureLoggedIn(ctx, pr)
	if err == nil || errdefs.CodeOf(err) != errdefs.CodeAuthRequired {
		t.Fatalf("err %v", err)
	}
}

func TestEnsureLoggedIn_nonInteractiveFails(t *testing.T) {
	r := testRunner(&loginExec{})
	r.opts.NonInteractive = true
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.ensureLoggedIn(ctx, nil)
	if err == nil || errdefs.CodeOf(err) != errdefs.CodeAuthRequired {
		t.Fatalf("err %v", err)
	}
}
