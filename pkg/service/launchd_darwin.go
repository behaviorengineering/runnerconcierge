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
			if !IsRunnerServiceName(name) {
				continue
			}
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			parsed := parseLaunchdPlist(data)
			cfg := parsed.ConfigPath
			if cfg == "" {
				cfg = configPathFromPlistLegacy(data)
			}
			cmd := SanitizeCommand(joinArgv(parsed.Argv))
			logon := login
			if strings.HasPrefix(dir, "/Library/LaunchDaemons") {
				logon = "root"
			}
			state, processUp := launchdStateAndProcess(ctx, exec, name)
			role := ClassifyRole(name, "launchd", cmd)
			out = append(out, Ownership{
				ServiceName: name,
				State:       state,
				LogonUser:   logon,
				ConfigPath:  cfg,
				Kind:        "launchd",
				UnitPath:    path,
				Command:     cmd,
				MatchReason: MatchReasonForLaunchd(name),
				Role:        role,
				ProcessUp:   processUp,
			})
		}
	}
	return out
}

func configPathFromPlistLegacy(data []byte) string {
	s := string(data)
	idx := strings.Index(s, "--config")
	if idx < 0 {
		return ""
	}
	rest := s[idx+len("--config"):]
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

func launchdStateAndProcess(ctx context.Context, exec gitexec.Exec, label string) (state string, processUp bool) {
	out, err := exec.Run(ctx, "launchctl", "list")
	if err != nil {
		return "unknown", false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, label) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pid := fields[0]
			processUp = pid != "-" && pid != "0"
			if fields[1] != "-" {
				return "running", processUp
			}
			return "stopped", processUp
		}
		return "stopped", false
	}
	return "unknown", false
}

func userHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
