package repair

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/inventory"
	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

// Options controls repair execution.
type Options struct {
	Exec             gitexec.Exec
	Prompter         prompt.Prompter
	ConfigPath       string
	ServiceName      string
	WindowsPassword  string
	AllowDestructive bool
	LoginUser        string
	MoveConfigToUser bool
	UseBrewServices  bool
}

// Result holds before/after inventory and backup path.
type Result struct {
	Before     *inventory.Report
	After      *inventory.Report
	BackupPath string
}

// Run uninstalls a bad service unit and reinstalls under the login user.
func Run(ctx context.Context, opts Options) (*Result, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errdefs.New("repair.Run", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if opts.Exec == nil {
		return nil, fmt.Errorf("repair: exec is nil")
	}
	if strings.TrimSpace(opts.ConfigPath) == "" {
		return nil, fmt.Errorf("repair: config path is required")
	}
	login := strings.TrimSpace(opts.LoginUser)
	if login == "" {
		u, err := service.LoginUser(opts.Exec)
		if err != nil {
			return nil, err
		}
		login = u
	}
	if !opts.AllowDestructive {
		if opts.Prompter == nil {
			return nil, fmt.Errorf("repair: pass --yes or run interactively")
		}
		msg := fmt.Sprintf("Rebind runner service for config %q as login user %q? This affects ALL [[runners]] in that config.", opts.ConfigPath, login)
		ok, err := opts.Prompter.Confirm(ctx, msg)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("repair: cancelled")
		}
	}
	before, err := inventory.Run(ctx, inventory.Options{Exec: opts.Exec, LoginUser: login, ExtraConfigPaths: []string{opts.ConfigPath}})
	if err != nil {
		return nil, err
	}
	svcMgr := service.New(opts.Exec)
	services, err := svcMgr.ListOwnership(ctx)
	if err != nil && !isUnsupported(err) {
		return nil, err
	}
	targetName, useBrew := resolveTarget(opts, services, before)
	if len(services) > 1 && strings.TrimSpace(opts.ServiceName) == "" && targetName == "" {
		return nil, fmt.Errorf("repair: multiple gitlab-runner services found; pass --service NAME")
	}
	if needsElevation(before, opts.ConfigPath) && !inventoryHasElevated(before) {
		return nil, errdefs.New("repair.Run", errdefs.CodeElevationRequired, "elevation required to repair this service", nil)
	}
	backup, err := snapshotConfig(opts.ConfigPath)
	if err != nil {
		return nil, err
	}
	cfgPath := opts.ConfigPath
	workDir := filepath.Dir(cfgPath)
	if opts.MoveConfigToUser && service.IsSystemConfigPath(cfgPath) {
		userCfg, userWork, _ := service.DefaultPaths()
		if userCfg != "" {
			if err := mergeConfig(cfgPath, userCfg); err != nil {
				return nil, err
			}
			cfgPath = userCfg
			workDir = userWork
		}
	}
	bin, err := opts.Exec.LookPath("gitlab-runner")
	if err != nil {
		return nil, errdefs.New("repair.Run", errdefs.CodeRunnerBinaryMissing, "gitlab-runner not on PATH", err)
	}
	un := service.UninstallOpts{
		BinaryPath:  bin,
		ConfigPath:  cfgPath,
		ServiceName: targetName,
		UseBrew:     useBrew || opts.UseBrewServices,
	}
	if err := svcMgr.Uninstall(ctx, un); err != nil {
		return nil, err
	}
	winUser := login
	if runtime.GOOS == "windows" {
		if strings.TrimSpace(opts.WindowsPassword) == "" {
			return nil, errdefs.New("repair.Run", errdefs.CodeServiceLogon, "Windows password required for service install", nil)
		}
		winUser = os.Getenv("USERNAME")
		if winUser == "" {
			winUser = login
		}
	}
	if err := svcMgr.Install(ctx, service.InstallOpts{
		BinaryPath:       bin,
		ConfigPath:       cfgPath,
		WorkingDirectory: workDir,
		WindowsUser:      winUser,
		WindowsPassword:  opts.WindowsPassword,
		UseBrewServices:  useBrew || opts.UseBrewServices,
	}); err != nil {
		return nil, fmt.Errorf("repair: install failed (backup at %s): %w", backup, err)
	}
	if err := svcMgr.Start(ctx); err != nil {
		return nil, fmt.Errorf("repair: start failed (backup at %s): %w", backup, err)
	}
	after, err := inventory.Run(ctx, inventory.Options{Exec: opts.Exec, LoginUser: login, ExtraConfigPaths: []string{cfgPath}})
	if err != nil {
		return nil, err
	}
	if inventory.HasBlocking(after) {
		return &Result{Before: before, After: after, BackupPath: backup}, fmt.Errorf("repair: blocking findings remain after repair")
	}
	return &Result{Before: before, After: after, BackupPath: backup}, nil
}

func snapshotConfig(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	backup := path + ".repair-bak-" + time.Now().UTC().Format("20060102T150405Z")
	if err := os.WriteFile(backup, data, 0o600); err != nil {
		return "", err
	}
	return backup, nil
}

func mergeConfig(from, to string) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
		return err
	}
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	if _, err := os.Stat(to); err == nil {
		existing, err := os.ReadFile(to)
		if err != nil {
			return err
		}
		merged := make([]byte, 0, len(existing)+1+len(data))
		merged = append(merged, existing...)
		merged = append(merged, '\n')
		merged = append(merged, data...)
		data = merged
	}
	return os.WriteFile(to, data, 0o600)
}

func resolveTarget(opts Options, services []service.Ownership, before *inventory.Report) (string, bool) {
	if strings.TrimSpace(opts.ServiceName) != "" {
		return strings.TrimSpace(opts.ServiceName), opts.UseBrewServices
	}
	for _, s := range services {
		if s.ConfigPath == opts.ConfigPath || strings.TrimSpace(s.ConfigPath) == "" {
			return s.ServiceName, s.Kind == "brew_services"
		}
	}
	if len(services) == 1 {
		return services[0].ServiceName, services[0].Kind == "brew_services"
	}
	return "", opts.UseBrewServices
}

func needsElevation(before *inventory.Report, configPath string) bool {
	if before == nil {
		return false
	}
	for _, f := range before.Findings {
		if f.ConfigPath == configPath && f.Code == errdefs.CodeElevationRequired {
			return true
		}
	}
	if service.IsSystemConfigPath(configPath) && runtime.GOOS == "windows" {
		return !before.Elevated
	}
	return false
}

func inventoryHasElevated(before *inventory.Report) bool {
	return before != nil && before.Elevated
}

func isUnsupported(err error) bool {
	var de *errdefs.Error
	if errors.As(err, &de) {
		return de.Code == errdefs.CodeUnsupportedOS
	}
	return false
}

// PrintResult writes before/after summaries.
func PrintResult(w io.Writer, res *Result) {
	if res == nil {
		return
	}
	_, _ = fmt.Fprintf(w, "backup: %s\n\n", res.BackupPath)
	_, _ = fmt.Fprintln(w, "before:")
	_ = inventory.Render(w, res.Before)
	_, _ = fmt.Fprintln(w, "\nafter:")
	_ = inventory.Render(w, res.After)
}
