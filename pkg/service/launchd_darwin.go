//go:build darwin

package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
)

func darwinLaunchdOwnerships(ctx context.Context, exec gitexec.Exec) []Ownership {
	var out []Ownership
	login, _ := LoginUser(exec)
	for _, dir := range []string{
		filepath.Join(userHome(), "Library", "LaunchAgents"),
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
			if !isRunnerPlistName(name) {
				continue
			}
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			cfg := configPathFromPlist(data)
			logon := login
			if strings.HasPrefix(dir, "/Library/LaunchDaemons") {
				logon = "root"
			}
			out = append(out, Ownership{
				ServiceName: name,
				State:       launchdState(ctx, exec, name),
				LogonUser:   logon,
				ConfigPath:  cfg,
				Kind:        "launchd",
			})
		}
	}
	return out
}

func isRunnerPlistName(name string) bool {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "runnerconcierge-e2e-") {
		return true
	}
	return strings.Contains(lower, "gitlab-runner")
}

func configPathFromPlist(data []byte) string {
	s := string(data)
	idx := strings.Index(s, "--config")
	if idx < 0 {
		return ""
	}
	rest := s[idx+len("--config"):]
	// plist often has </string> between key and value; grab next path-like token
	for _, part := range strings.FieldsFunc(rest, func(r rune) bool {
		return r == '<' || r == '>' || r == '\n' || r == '\t'
	}) {
		part = strings.TrimSpace(part)
		if strings.HasSuffix(part, ".toml") {
			return part
		}
	}
	return ""
}

func launchdState(ctx context.Context, exec gitexec.Exec, label string) string {
	out, err := exec.Run(ctx, "launchctl", "list")
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, label) {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[1] != "-" {
				return "running"
			}
			return "stopped"
		}
	}
	return "unknown"
}

func userHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
