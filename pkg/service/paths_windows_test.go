//go:build windows

package service

import (
	"path/filepath"
	"testing"
)

func TestDefaultPathsUseLocalAppData(t *testing.T) {
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)

	config, work, binary := DefaultPaths()
	wantWork := filepath.Join(localAppData, "GitLab-Runner")
	if work != wantWork {
		t.Fatalf("work path = %q, want %q", work, wantWork)
	}
	wantConfig := filepath.Join(wantWork, "config.toml")
	if config != wantConfig {
		t.Fatalf("config path = %q, want %q", config, wantConfig)
	}
	if IsSystemConfigPath(config) {
		t.Fatalf("config path %q must be user-scoped", config)
	}
	if binary != "" {
		t.Fatalf("binary path = %q, want empty", binary)
	}
}
