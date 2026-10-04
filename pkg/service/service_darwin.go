//go:build darwin

package service

import (
	"context"
	"os"
	"strings"
	"time"

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

func (m *darwinManager) Uninstall(ctx context.Context, opts UninstallOpts) error {
	if _, ok := ctx.Deadline(); !ok {
		return errdefs.New("service.Uninstall", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if opts.UseBrew {
		_, err := m.exec.Run(ctx, "brew", "services", "stop", "gitlab-runner")
		if err != nil {
			return errdefs.New("service.Uninstall", errdefs.CodeServiceStart, "brew services stop", err)
		}
	}
	bin := opts.BinaryPath
	if bin == "" {
		bin, _ = m.exec.LookPath("gitlab-runner")
	}
	if bin == "" {
		return errdefs.New("service.Uninstall", errdefs.CodeRunnerBinaryMissing, "gitlab-runner not on PATH", nil)
	}
	_, _ = m.exec.Run(ctx, bin, "stop")
	args := []string{"uninstall"}
	_, err := m.exec.Run(ctx, bin, args...)
	if err != nil {
		return errdefs.New("service.Uninstall", errdefs.CodeServiceStart, "gitlab-runner uninstall", err)
	}
	return nil
}

func (m *darwinManager) ListOwnership(ctx context.Context) ([]Ownership, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errdefs.New("service.ListOwnership", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	var out []Ownership
	if _, err := m.exec.LookPath("brew"); err == nil {
		info, err := m.exec.Run(ctx, "brew", "services", "list")
		if err == nil {
			for _, line := range strings.Split(string(info), "\n") {
				if !strings.Contains(line, "gitlab-runner") {
					continue
				}
				fields := strings.Fields(line)
				state := "unknown"
				if len(fields) >= 2 {
					state = fields[1]
				}
				login, _ := LoginUser(m.exec)
				out = append(out, Ownership{
					ServiceName: "gitlab-runner",
					State:       state,
					LogonUser:   login,
					Kind:        "brew_services",
				})
			}
		}
	}
	st, _ := m.Status(ctx)
	if strings.Contains(strings.ToLower(st), "is running") || strings.Contains(strings.ToLower(st), "not installed") {
		cfg, _, _ := DefaultPaths()
		login, _ := LoginUser(m.exec)
		procUser := darwinProcessUser(ctx, m.exec)
		if len(out) == 0 {
			out = append(out, Ownership{
				ServiceName: "gitlab-runner",
				State:       classifyDarwinStatus(st),
				LogonUser:   firstNonEmpty(procUser, login),
				ConfigPath:  cfg,
				Kind:        "launchd",
			})
		}
	}
	return out, nil
}

func classifyDarwinStatus(status string) string {
	lower := strings.ToLower(status)
	if strings.Contains(lower, "is running") {
		return "running"
	}
	if strings.Contains(lower, "not installed") {
		return "stopped"
	}
	return "unknown"
}

func darwinProcessUser(ctx context.Context, exec gitexec.Exec) string {
	out, err := exec.Run(ctx, "pgrep", "-lf", "gitlab-runner")
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	if line == "" {
		return ""
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	pid := fields[0]
	psOut, err := exec.Run(ctx, "ps", "-o", "user=", "-p", pid)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(psOut))
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}

func (m *darwinManager) execUser() string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := m.exec.Run(ctx, "whoami")
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
