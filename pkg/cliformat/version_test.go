package cliformat

import "testing"

func TestCompactVersion(t *testing.T) {
	if got := CompactVersion("Version:      19.4.0\nGit revision: ac11717a\n"); got != "19.4.0" {
		t.Fatalf("got %q", got)
	}
	if got := CompactVersion(""); got != "(not found)" {
		t.Fatalf("got %q", got)
	}
	if got := CompactVersion("glab 1.118.0 (570955d42)"); got != "glab 1.118.0 (570955d42)" {
		t.Fatalf("got %q", got)
	}
}
