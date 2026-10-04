//go:build e2e_live && (darwin || windows)

package e2elive

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/internal/e2elive/fixture"
	"github.com/behaviorengineering/runnerconcierge/pkg/inventory"
)

func TestObserveStatus(t *testing.T) {
	mode := os.Getenv("E2E_LIVE_MODE")
	if mode == "repair-prod" && os.Getenv("E2E_LIVE_ALLOW_PROD") != "1" {
		skipOrFatal(t, "repair-prod requires E2E_LIVE_ALLOW_PROD=1")
	}
	if _, err := gitexec.New().LookPath("gitlab-runner"); err != nil {
		skipOrFatal(t, "gitlab-runner not on PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	rep, err := inventory.Run(ctx, inventory.Options{Exec: gitexec.New()})
	if err != nil {
		t.Fatal(err)
	}
	if rep.GOOS == "" {
		t.Fatal("expected GOOS")
	}
}

func TestFixtureLifecycle(t *testing.T) {
	if os.Getenv("E2E_LIVE_MODE") != "fixture" {
		skipOrFatal(t, "E2E_LIVE_MODE=fixture required")
	}
	exec := gitexec.New()
	cliPath := buildCLI(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	st, err := fixture.Run(ctx, fixture.Options{
		Exec:    exec,
		CLIPath: cliPath,
	})
	if errors.Is(err, fixture.ErrSkipped) {
		if os.Getenv("E2E_LIVE_MODE") == "fixture" || os.Getenv("E2E_LIVE_REQUIRE") == "1" {
			t.Fatal(err.Error())
		}
		t.Skip(err.Error())
	}
	if st != nil {
		t.Cleanup(func() {
			disposeCtx, disposeCancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer disposeCancel()
			if derr := fixture.Dispose(disposeCtx, exec, st); derr != nil {
				t.Errorf("fixture dispose: %v", derr)
			}
		})
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegisterLifecycle(t *testing.T) {
	if !fixture.WantRegister() {
		t.Skip(fixture.SkipRegisterReason())
	}
	if reason := fixture.SkipRegisterReason(); reason != "" {
		skipOrFatal(t, reason)
	}
	if _, err := gitexec.New().LookPath("gitlab-runner"); err != nil {
		skipOrFatal(t, "gitlab-runner not on PATH")
	}
	exec := gitexec.New()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	st, err := fixture.RunRegister(ctx, fixture.Options{Exec: exec})
	if st != nil {
		t.Cleanup(func() {
			disposeCtx, disposeCancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer disposeCancel()
			if derr := fixture.DisposeRegister(disposeCtx, exec, st); derr != nil {
				t.Errorf("register dispose: %v", derr)
			}
		})
	}
	if err != nil {
		t.Fatal(err)
	}
}

func buildCLI(t *testing.T) string {
	dir := t.TempDir()
	out := filepath.Join(dir, "runnerconcierge")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", out, "./cmd/runnerconcierge")
	cmd.Dir = moduleRoot(t)
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build cli: %v\n%s", err, outBytes)
	}
	return out
}

func moduleRoot(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("go.mod not found")
		}
		wd = parent
	}
}

func skipOrFatal(t *testing.T, reason string) {
	if os.Getenv("E2E_LIVE_REQUIRE") == "1" {
		t.Fatal(reason)
	}
	t.Skip(reason)
}
