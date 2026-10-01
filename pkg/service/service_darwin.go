//go:build darwin

package service

import (
	"context"
	"os"
	"strings"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

type darwinManager struct {
	exec gitexec.Exec
}

func newPlatformManager(exec gitexec.Exec) Manager {
	return &darwinManager{exec: exec}
}

func (m *darwinManager) Install(ctx context.Context, opts InstallOpts) error {
	if _, ok := ctx.Deadline(); !ok {
		return errdefs.New("service.Install", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if opts.UseBrewServices {
		_, err := m.exec.Run(ctx, "brew", "services", "start", "gitlab-runner")
		if err != nil {
			return errdefs.New("service.Install", errdefs.CodeServiceStart, "brew services start", err)
		}
		return nil
	}
	args := []string{"install", "--user", m.execUser()}
	if opts.WorkingDirectory != "" {
		args = append(args, "--working-directory", opts.WorkingDirectory)
	}
	if opts.ConfigPath != "" {
		args = append(args, "--config", opts.ConfigPath)
	}
	_, err := m.exec.Run(ctx, "gitlab-runner", args...)
	if err != nil {
		return errdefs.New("service.Install", errdefs.CodeServiceStart, "gitlab-runner install", err)
	}
	return nil
}

func (m *darwinManager) Start(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return errdefs.New("service.Start", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if _, err := m.exec.LookPath("brew"); err == nil {
		_, err := m.exec.Run(ctx, "brew", "services", "restart", "gitlab-runner")
		return err
	}
	_, err := m.exec.Run(ctx, "gitlab-runner", "start")
	return err
}

func (m *darwinManager) Status(ctx context.Context) (string, error) {
	out, err := m.exec.Run(ctx, "gitlab-runner", "status")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (m *darwinManager) execUser() string {
	out, err := m.exec.Run(context.Background(), "whoami")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// EnsureSingleProcess is a no-op placeholder on darwin; doctor handles conflicts.
func EnsureSingleProcess(ctx context.Context, exec gitexec.Exec) error {
	return nil
}

// DefaultPaths returns macOS runner paths.
func DefaultPaths() (config, work, binary string) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	cfg := home + "/.gitlab-runner/config.toml"
	return cfg, home, "/usr/local/bin/gitlab-runner"
}
