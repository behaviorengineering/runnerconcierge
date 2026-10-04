package detect

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderDoctorSections(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, &Report{
		GOOS:      "darwin",
		GOARCH:    "arm64",
		Username:  "hector",
		GitOK:     true,
		RunnerVer: "Version:      19.4.0\nGit revision: ac11717a\n",
		GlabVer:   "glab 1.118.0",
		Issues: []Issue{{
			Code:    "git_missing",
			Message: "example",
			Block:   false,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Host", "Tools", "Issues", "gitlab-runner:         19.4.0"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Git revision") {
		t.Fatal("version banner should be compact")
	}
}

func TestRenderJSONKeepsFullVersion(t *testing.T) {
	raw := "Version:      19.4.0\nGit revision: ac11717a\n"
	var buf bytes.Buffer
	if err := RenderJSON(&buf, &Report{RunnerVer: raw}); err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.RunnerVer != raw {
		t.Fatalf("json version: %q", decoded.RunnerVer)
	}
}

func TestHasBlocking(t *testing.T) {
	if HasBlocking(&Report{Issues: []Issue{{Block: false}}}) {
		t.Fatal("expected false")
	}
	if !HasBlocking(&Report{Issues: []Issue{{Block: true}}}) {
		t.Fatal("expected true")
	}
}
