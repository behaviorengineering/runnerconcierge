package repair

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
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

func TestMergeConfigSamePathDoesNotDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	want := []byte("[[runners]]\nname = \"runner\"\n")
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := mergeConfig(path, path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("same-path merge changed config: got %q, want %q", got, want)
	}
}

func TestHasTargetService(t *testing.T) {
	services := []service.Ownership{{ServiceName: "gitlab-runner"}}
	if hasTargetService(nil, "gitlab-runner") {
		t.Fatal("reported a target service in an empty service list")
	}
	if hasTargetService(services, "") {
		t.Fatal("reported a target service for an empty target name")
	}
	if !hasTargetService(services, "GITLAB-RUNNER") {
		t.Fatal("did not match the discovered target service")
	}
}

func TestHasRunningServiceForConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	services := []service.Ownership{{
		ServiceName: "gitlab-runner",
		ConfigPath:  configPath,
		Role:        service.RoleSupervisor,
		ProcessUp:   true,
	}}
	if !hasRunningServiceForConfig(services, "gitlab-runner", configPath) {
		t.Fatal("expected matching running service")
	}
	if hasRunningServiceForConfig(services, "gitlab-runner", configPath+".other") {
		t.Fatal("accepted service with a different config path")
	}
	services[0].ProcessUp = false
	if hasRunningServiceForConfig(services, "gitlab-runner", configPath) {
		t.Fatal("accepted a stopped service")
	}
}
