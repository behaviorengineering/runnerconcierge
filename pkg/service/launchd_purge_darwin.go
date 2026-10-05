//go:build darwin

package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
)

// ForceRemoveLaunchdPlist unloads and deletes a gitlab-runner launchd plist when uninstall left it behind.
func ForceRemoveLaunchdPlist(ctx context.Context, exec gitexec.Exec, label string) {
	label = strings.TrimSpace(label)
	if label == "" || exec == nil {
		return
	}
	home, _ := os.UserHomeDir()
	for _, spec := range []struct {
		dir  string
		sudo bool
	}{
		{filepath.Join(home, "Library", "LaunchAgents"), false},
		{"/Library/LaunchAgents", true},
		{"/Library/LaunchDaemons", true},
	} {
		plist := filepath.Join(spec.dir, label+".plist")
		if _, err := os.Stat(plist); err != nil {
			continue
		}
		darwinLaunchctlBootout(ctx, exec, label, spec.sudo)
		if spec.sudo {
			_ = darwinRunSudo(ctx, exec, "/bin/rm", "-f", plist)
		} else {
			_ = os.Remove(plist)
		}
	}
}

// ForceRemoveLaunchdPlistsWithPrefix deletes leftover fixture launchd units whose names share prefix.
func ForceRemoveLaunchdPlistsWithPrefix(ctx context.Context, exec gitexec.Exec, prefix string) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || exec == nil {
		return
	}
	home, _ := os.UserHomeDir()
	seen := map[string]struct{}{}
	for _, dir := range []string{
		filepath.Join(home, "Library", "LaunchAgents"),
		"/Library/LaunchAgents",
		"/Library/LaunchDaemons",
	} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".plist") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".plist")
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			ForceRemoveLaunchdPlist(ctx, exec, name)
		}
	}
}

func darwinLaunchctlBootout(ctx context.Context, exec gitexec.Exec, label string, system bool) {
	if system {
		_ = darwinRunSudo(ctx, exec, "launchctl", "bootout", "system/"+label)
		return
	}
	out, err := exec.Run(ctx, "id", "-u")
	if err != nil {
		return
	}
	uid := strings.TrimSpace(string(out))
	if uid == "" {
		return
	}
	_, _ = exec.Run(ctx, "launchctl", "bootout", "gui/"+uid+"/"+label)
	_, _ = exec.Run(ctx, "launchctl", "bootout", "user/"+uid+"/"+label)
}
