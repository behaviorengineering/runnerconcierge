package inventory

import (
	"path/filepath"
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
