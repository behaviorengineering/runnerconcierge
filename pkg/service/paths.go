package service

import (
	"os"
	"runtime"
	"strings"
)

// CandidateConfigPaths returns known gitlab-runner config.toml locations for the OS.
func CandidateConfigPaths() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			"C:\\GitLab-Runner\\config.toml",
			os.ExpandEnv("${LOCALAPPDATA}\\GitLab-Runner\\config.toml"),
		}
	default:
		home, _ := os.UserHomeDir()
		return []string{
			home + "/.gitlab-runner/config.toml",
			"/etc/gitlab-runner/config.toml",
		}
	}
}

// IsSystemConfigPath reports whether path is a system-scoped runner config.
func IsSystemConfigPath(path string) bool {
	p := strings.TrimSpace(path)
	switch runtime.GOOS {
	case "windows":
		return strings.HasPrefix(strings.ToLower(p), "c:\\gitlab-runner\\")
	case "darwin":
		return p == "/etc/gitlab-runner/config.toml" || strings.HasPrefix(p, "/etc/gitlab-runner/")
	default:
		return p == "/etc/gitlab-runner/config.toml"
	}
}
