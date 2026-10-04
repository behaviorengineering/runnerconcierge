package inventory

import (
	"bytes"
	"strings"
	"testing"

	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

func TestRenderSectionsAndCompactVersion(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, &Report{
		GOOS:      "darwin",
		GOARCH:    "arm64",
		LoginUser: "hector",
		RunnerVer: "Version:      19.4.0\nGit revision: ac11717a\n",
		Configs: []ConfigReport{{
			Path:     "/Users/hector/.gitlab-runner/config.toml",
			Readable: true,
			Runners:  []RunnerEntry{{Name: "macos-dualsubstrate", URL: "https://gitlab.com/", Executor: "docker"}},
		}},
		Services: []service.Ownership{
			{ServiceName: "sh.brew.gitlab-runner", Kind: "launchd", State: "running", LogonUser: "hector"},
			{ServiceName: "gitlab-runner", Kind: "brew_services", State: "started", LogonUser: "hector"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"Host",
		"gitlab-runner:       19.4.0",
		"Registered runners",
		"Services",
		"brew_services  (Homebrew",
		"same Homebrew runner",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Git revision") {
		t.Fatal("full version dump should be compacted")
	}
	if !strings.Contains(out, "\n\n") {
		t.Fatal("expected blank lines between sections")
	}
}
