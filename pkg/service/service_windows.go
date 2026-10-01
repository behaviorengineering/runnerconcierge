//go:build windows

package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

const (
	defaultRoot       = "C:\\GitLab-Runner"
	defaultConfigPath = "C:\\GitLab-Runner\\config.toml"
)

type windowsManager struct {
	exec gitexec.Exec
}

func newPlatformManager(exec gitexec.Exec) Manager {
	return &windowsManager{exec: exec}
}

func (m *windowsManager) Install(ctx context.Context, opts InstallOpts) error {
	if _, ok := ctx.Deadline(); !ok {
		return errdefs.New("service.Install", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	user := strings.TrimSpace(opts.WindowsUser)
	if user == "" {
		user = os.Getenv("USERNAME")
	}
	if strings.EqualFold(user, "SYSTEM") || strings.Contains(strings.ToUpper(user), "LOCALSYSTEM") {
		return errdefs.New("service.Install", errdefs.CodeServiceLogon, "refuse LocalSystem; use the interactive login user", nil)
	}
	if strings.TrimSpace(opts.WindowsPassword) == "" {
		return errdefs.New("service.Install", errdefs.CodeServiceLogon, "Windows password required for user service account", nil)
	}
	bin := opts.BinaryPath
	if bin == "" {
		bin, _ = m.exec.LookPath("gitlab-runner")
	}
	cfg := opts.ConfigPath
	if cfg == "" {
		cfg = defaultConfigPath
	}
	work := opts.WorkingDirectory
	if work == "" {
		work = defaultRoot
	}
	_ = os.MkdirAll(work, 0o755)
	domainUser := formatWindowsUser(user)
	args := []string{
		"install",
		"--user", domainUser,
		"--password", opts.WindowsPassword,
		"--config", cfg,
		"--working-directory", work,
	}
	_, err := m.exec.Run(ctx, bin, args...)
	if err != nil {
		if isLogonFailure(err) {
			return errdefs.New("service.Install", errdefs.CodeServiceLogon, "logon failure (1069); grant SeServiceLogonRight", err)
		}
		return errdefs.New("service.Install", errdefs.CodeServiceStart, "gitlab-runner install", err)
	}
	_ = hardenACL(ctx, m.exec, work)
	return nil
}

func (m *windowsManager) Start(ctx context.Context) error {
	_, err := m.exec.Run(ctx, "gitlab-runner", "start")
	return err
}

func (m *windowsManager) Status(ctx context.Context) (string, error) {
	out, err := m.exec.Run(ctx, "gitlab-runner", "status")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func formatWindowsUser(user string) string {
	if strings.Contains(user, "\\") {
		return user
	}
	return ".\\" + user
}

func isLogonFailure(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "1069") || strings.Contains(strings.ToLower(err.Error()), "logon")
}

func hardenACL(ctx context.Context, exec gitexec.Exec, dir string) error {
	_, err := exec.Run(ctx, "icacls", dir, "/inheritance:r")
	if err != nil {
		return err
	}
	user := os.Getenv("USERNAME")
	if user != "" {
		_, _ = exec.Run(ctx, "icacls", dir, "/grant:r", fmt.Sprintf("%s:(OI)(CI)F", user))
	}
	_, _ = exec.Run(ctx, "icacls", dir, "/grant:r", "SYSTEM:(OI)(CI)F")
	return nil
}

// DefaultPaths returns Windows runner paths.
func DefaultPaths() (config, work, binary string) {
	return defaultConfigPath, defaultRoot, ""
}

// EnsureSingleProcess stops duplicate managers when possible.
func EnsureSingleProcess(ctx context.Context, exec gitexec.Exec) error {
	_, _ = exec.Run(ctx, "powershell", "-NoProfile", "-Command", "Get-Process gitlab-runner -ErrorAction SilentlyContinue | Stop-Process -Force")
	return nil
}
