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
