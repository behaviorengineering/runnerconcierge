package wizard

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/behaviorengineering/runnerconcierge/internal/config"
)

type onlineVerifyExec struct{}

func (onlineVerifyExec) LookPath(name string) (string, error) { return name, nil }
func (onlineVerifyExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return onlineVerifyExec{}.Run(ctx, name, args...)
}
func (onlineVerifyExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "glab" && len(args) == 2 && args[0] == "api" && args[1] == "runners/9" {
		return json.Marshal(map[string]any{"online": true})
	}
	return nil, nil
}

func TestVerifyRunnerOnlineOnGitLab_ok(t *testing.T) {
	cfg, _ := config.Load("")
	var msg string
	r := &Runner{
		cfg:  cfg,
		exec: onlineVerifyExec{},
		out:  func(s string) { msg = s },
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.verifyRunnerOnlineOnGitLab(ctx, 9); err != nil {
		t.Fatal(err)
	}
	if msg == "" || !strings.Contains(msg, "verified online") {
		t.Fatalf("out %q", msg)
	}
}
