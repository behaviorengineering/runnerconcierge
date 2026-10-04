package repair

import (
	"context"
	"testing"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
)

type seqExec struct {
	gitexec.Exec
	calls []string
}

func (s *seqExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	s.calls = append(s.calls, name+":"+join(args))
	if name == "whoami" {
		return []byte("testuser\n"), nil
	}
	return []byte("ok"), nil
}

func (s *seqExec) LookPath(name string) (string, error) {
	if name == "gitlab-runner" {
		return "/usr/local/bin/gitlab-runner", nil
	}
	return "", nil
}

func join(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}

func TestRepairRequiresConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := Run(ctx, Options{Exec: &seqExec{}, AllowDestructive: true})
	if err == nil {
		t.Fatal("expected error")
	}
}
