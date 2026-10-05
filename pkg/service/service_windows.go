//go:build windows

package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
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
	defaultConfig, defaultWork := windowsDefaultPaths()
	cfg := opts.ConfigPath
	if cfg == "" {
		cfg = defaultConfig
	}
	work := opts.WorkingDirectory
	if work == "" {
		work = defaultWork
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
	if strings.TrimSpace(opts.ServiceName) != "" {
		args = append(args, "--service", strings.TrimSpace(opts.ServiceName))
	}
	_, err := m.exec.Run(ctx, bin, args...)
	if err != nil {
		if isLogonFailure(err) {
			_ = grantSeServiceLogonRight(ctx, m.exec, domainUser)
			return errdefs.New("service.Install", errdefs.CodeServiceLogon, "logon failure (1069); grant SeServiceLogonRight and retry", err)
		}
		return errdefs.New("service.Install", errdefs.CodeServiceStart, "gitlab-runner install", err)
	}
	_ = hardenACL(ctx, m.exec, work)
	return nil
}

func (m *windowsManager) Start(ctx context.Context, opts StartOpts) error {
	if _, ok := ctx.Deadline(); !ok {
		return errdefs.New("service.Start", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	args := []string{"start"}
	if strings.TrimSpace(opts.ServiceName) != "" {
		args = append(args, "--service", strings.TrimSpace(opts.ServiceName))
	}
	_, err := m.exec.Run(ctx, "gitlab-runner", args...)
	if err != nil {
		return errdefs.New("service.Start", errdefs.CodeServiceStart, "gitlab-runner start", err)
	}
	return nil
}

func (m *windowsManager) Stop(ctx context.Context, opts StopOpts) error {
	if _, ok := ctx.Deadline(); !ok {
		return errdefs.New("service.Stop", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	args := []string{"stop"}
	if strings.TrimSpace(opts.ServiceName) != "" {
		args = append(args, "--service", strings.TrimSpace(opts.ServiceName))
	}
	_, err := m.exec.Run(ctx, "gitlab-runner", args...)
	if err != nil && !windowsStopBenign(err) {
		return errdefs.New("service.Stop", errdefs.CodeServiceStart, "gitlab-runner stop", err)
	}
	return nil
}

func windowsStopBenign(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not installed") || strings.Contains(msg, "not running")
}

func (m *windowsManager) Status(ctx context.Context) (string, error) {
	out, err := m.exec.Run(ctx, "gitlab-runner", "status")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (m *windowsManager) Uninstall(ctx context.Context, opts UninstallOpts) error {
	if _, ok := ctx.Deadline(); !ok {
		return errdefs.New("service.Uninstall", errdefs.CodeMissingDeadline, "context missing deadline", nil)
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
	if strings.TrimSpace(opts.ServiceName) != "" {
		args = append(args, "--service", strings.TrimSpace(opts.ServiceName))
	}
	_, err := m.exec.Run(ctx, bin, args...)
	if err != nil {
		return errdefs.New("service.Uninstall", errdefs.CodeServiceStart, "gitlab-runner uninstall", err)
	}
	return nil
}

func (m *windowsManager) ListOwnership(ctx context.Context) ([]Ownership, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errdefs.New("service.ListOwnership", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	script := `
Get-CimInstance Win32_Service | Where-Object {
  $_.Name -like 'gitlab-runner*' -or ($_.PathName -and $_.PathName -match 'gitlab-runner')
} | ForEach-Object {
  [PSCustomObject]@{
    Name=$_.Name
    State=$_.State
    StartName=$_.StartName
    PathName=$_.PathName
  }
} | ConvertTo-Json -Compress
`
	psOut, err := m.exec.Run(ctx, "powershell", "-NoProfile", "-Command", script)
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(psOut))
	var list []Ownership
	if raw != "" && raw != "null" {
		list, err = parseWindowsOwnershipJSON(raw)
		if err != nil {
			return nil, err
		}
	}
	if task := windowsCleanupTaskOwnership(ctx, m.exec); task != nil {
		list = append(list, *task)
	}
	return list, nil
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
	config, work = windowsDefaultPaths()
	return config, work, ""
}

func windowsDefaultPaths() (config, work string) {
	work = filepath.Join(os.Getenv("LOCALAPPDATA"), "GitLab-Runner")
	return filepath.Join(work, "config.toml"), work
}

// EnsureSingleProcess stops duplicate managers when possible.
func grantSeServiceLogonRight(ctx context.Context, exec gitexec.Exec, user string) error {
	if user == "" {
		return nil
	}
	script := fmt.Sprintf(`
$u = "%s"
secedit /export /cfg $env:TEMP\secpol.cfg | Out-Null
(Get-Content $env:TEMP\secpol.cfg) -replace 'SeServiceLogonRight = .*', ('SeServiceLogonRight = ' + $u) | Set-Content $env:TEMP\secpol.cfg
secedit /configure /db secedit.sdb /cfg $env:TEMP\secpol.cfg /areas USER_RIGHTS | Out-Null
`, strings.ReplaceAll(user, `"`, `\"`))
	_, err := exec.Run(ctx, "powershell", "-NoProfile", "-Command", script)
	return err
}

func EnsureSingleProcess(ctx context.Context, exec gitexec.Exec) error {
	_, _ = exec.Run(ctx, "powershell", "-NoProfile", "-Command", "Get-Process gitlab-runner -ErrorAction SilentlyContinue | Stop-Process -Force")
	return nil
}
