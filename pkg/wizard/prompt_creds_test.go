package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
)

type recordingPrompter struct {
	passwords []string
	passwordN int
}

func (r *recordingPrompter) Confirm(ctx context.Context, title string) (bool, error) {
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

func TestPromptCredentialsAsksGLRTFirst(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	pr := &recordingPrompter{passwords: []string{"glrt-paste"}}
	r := &Runner{
		opts: Options{Keyring: mem, TagList: []string{"homelab-mac"}},
	}
	if err := r.promptCredentials(context.Background(), pr, 0, nil); err != nil {
		t.Fatal(err)
	}
	if r.opts.RunnerToken != "glrt-paste" {
		t.Fatalf("token %q", r.opts.RunnerToken)
	}
	if pr.passwordN != 1 {
		t.Fatalf("password calls %d", pr.passwordN)
	}
	got, err := config.LoadRunnerTokenByTag("homelab-mac", mem)
	if err != nil || got != "glrt-paste" {
		t.Fatalf("tag store %q err %v", got, err)
	}
}

func TestPromptCredentialsSkipsWhenKeyringSet(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	if err := config.StoreRunnerTokenByTag("lab", "glrt-saved", mem); err != nil {
		t.Fatal(err)
	}
	pr := &recordingPrompter{}
	r := &Runner{opts: Options{Keyring: mem, TagList: []string{"lab"}}}
	if err := r.promptCredentials(context.Background(), pr, 0, nil); err != nil {
		t.Fatal(err)
	}
	if pr.passwordN != 0 {
		t.Fatal("should not prompt")
	}
	if r.opts.RunnerToken != "glrt-saved" {
		t.Fatalf("token %q", r.opts.RunnerToken)
	}
}

func TestPromptCredentialsEmptyDoesNotStore(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	pr := &recordingPrompter{passwords: []string{""}}
	r := &Runner{opts: Options{Keyring: mem, TagList: []string{"lab"}}}
	if err := r.promptCredentials(context.Background(), pr, 0, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := config.LoadRunnerTokenByTag("lab", mem)
	if got != "" {
		t.Fatalf("unexpected tag store %q", got)
	}
}

func TestPromptCredentialsGLRTTitle(t *testing.T) {
	var ni prompt.NonInteractive
	_, err := ni.Password(context.Background(), glrtPasswordTitle)
	if err == nil || !strings.Contains(glrtPasswordTitle, "glrt") {
		t.Fatalf("title %q err %v", glrtPasswordTitle, err)
	}
}
