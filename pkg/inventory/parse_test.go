package inventory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigFile(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "inventory", "two_runners.toml")
	entries, err := parseConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 runners, got %d", len(entries))
	}
	if entries[0].Name != "home-a" {
		t.Fatalf("unexpected name %q", entries[0].Name)
	}
}

func TestParseConfigFile_gitlabID(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "inventory", "runner_with_id.toml")
	entries, err := parseConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].GitLabID != 42 {
		t.Fatalf("gitlab id: %+v", entries[0])
	}
}

func TestParseConfigFile_ignoresNestedRunnerSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	data := []byte(`[[runners]]
id = 42
name = "nested-runner"
url = "https://gitlab.example.com"
token = "secret-token"
executor = "docker"
token_obtained_at = 2026-10-05T00:00:00Z

[runners.docker]
image = "alpine:latest"
privileged = true

[runners.cache.s3]
ServerAddress = "s3.example.com"
BucketName = "runner-cache"
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := parseConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 runner, got %d", len(entries))
	}
	if entries[0].Name != "nested-runner" || entries[0].GitLabID != 42 || entries[0].Executor != "docker" {
		t.Fatalf("unexpected runner entry: %+v", entries[0])
	}
}

func TestConfigDiagnosticOmitsConfigValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	data := []byte("token = \"secret-token\"\ninvalid = [")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := parseConfigFile(path); err == nil {
		t.Fatal("expected TOML decode error")
	} else {
		diagnostic := configDiagnostic(err)
		if !strings.HasPrefix(diagnostic, "TOML decode failed at line 2, column ") {
			t.Fatalf("unexpected diagnostic %q", diagnostic)
		}
		if strings.Contains(diagnostic, "secret-token") {
			t.Fatalf("diagnostic exposed config value: %q", diagnostic)
		}
	}
}

func TestConfigDiagnosticReportsRunnerShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	data := []byte("runners = { name = \"secret-token\" }")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := parseConfigFile(path); err == nil {
		t.Fatal("expected invalid runners shape")
	} else {
		diagnostic := configDiagnostic(err)
		if diagnostic != "runners field is not an array of tables" {
			t.Fatalf("unexpected diagnostic %q", diagnostic)
		}
		if strings.Contains(diagnostic, "secret-token") {
			t.Fatalf("diagnostic exposed config value: %q", diagnostic)
		}
	}
}
