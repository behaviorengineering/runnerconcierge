package install

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

// Tool names.
const (
	ToolGitLabRunner = "gitlab-runner"
	ToolGlab         = "glab"
)

// Installer ensures CLI tools exist.
type Installer struct {
	Exec gitexec.Exec
}

// New panics on nil exec.
func New(exec gitexec.Exec) *Installer {
	if exec == nil {
		panic("install: exec is nil")
	}
	return &Installer{Exec: exec}
}

// Ensure looks up a tool or installs it when allowInstall is true.
func (i *Installer) Ensure(ctx context.Context, tool string, allowInstall bool) (string, error) {
	if i == nil {
		return "", fmt.Errorf("install: installer is nil")
	}
	if _, ok := ctx.Deadline(); !ok {
		return "", errdefs.New("Ensure", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	path, err := i.Exec.LookPath(tool)
	if err == nil {
		return path, nil
	}
	if !allowInstall {
		return "", errdefs.New("Ensure", errdefs.CodeInstallRefused, tool+" missing; re-run with install consent", err)
	}
	switch tool {
	case ToolGitLabRunner:
		return i.installRunner(ctx)
	case ToolGlab:
		return i.installGlab(ctx)
	default:
		return "", fmt.Errorf("install: unknown tool %s", tool)
	}
}

func (i *Installer) installRunner(ctx context.Context) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		_, err := i.Exec.Run(ctx, "brew", "install", "gitlab-runner")
		if err != nil {
			return "", errdefs.New("installRunner", errdefs.CodeDownloadFailed, "brew install gitlab-runner", err)
		}
	case "windows":
		_, err := i.Exec.Run(ctx, "winget", "install", "--id", "Gitlab.Runner", "-e", "--accept-package-agreements", "--accept-source-agreements")
		if err != nil {
			return "", errdefs.New("installRunner", errdefs.CodeDownloadFailed, "winget install Gitlab.Runner", err)
		}
	default:
		return "", errdefs.New("installRunner", errdefs.CodeUnsupportedOS, "automatic install not supported", nil)
	}
	return i.Exec.LookPath(ToolGitLabRunner)
}

func (i *Installer) installGlab(ctx context.Context) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		_, err := i.Exec.Run(ctx, "brew", "install", "glab")
		if err != nil {
			return "", err
		}
	case "windows":
		_, err := i.Exec.Run(ctx, "winget", "install", "--id", "GLab.GLab", "-e", "--accept-package-agreements", "--accept-source-agreements")
		if err != nil {
			return "", err
		}
	default:
		return "", errdefs.New("installGlab", errdefs.CodeUnsupportedOS, "automatic install not supported", nil)
	}
	return i.Exec.LookPath(ToolGlab)
}

// LongPathsWindows sets git core.longpaths when on Windows.
func LongPathsWindows(ctx context.Context, exec gitexec.Exec) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	if exec == nil {
		return fmt.Errorf("install: exec is nil")
	}
	_, err := exec.Run(ctx, "git", "config", "--system", "core.longpaths", "true")
	if err != nil && !strings.Contains(err.Error(), "permission") {
		return err
	}
	return nil
}
