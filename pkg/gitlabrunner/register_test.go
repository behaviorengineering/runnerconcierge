package gitlabrunner

import (
	"strings"
	"testing"
)

func TestBuildRegisterArgv_noServerTags(t *testing.T) {
	args, err := BuildRegisterArgv(RegisterArgs{
		URL:      "https://gitlab.com/",
		Token:    "glrt-test",
		Name:     "host",
		Executor: "shell",
		Shell:    "pwsh",
		ConfigPath: "C:\\GitLab-Runner\\config.toml",
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--tag-list") || strings.Contains(joined, "--run-untagged") {
		t.Fatalf("server-side flags must not appear in register argv: %v", args)
	}
	if !strings.Contains(joined, "--shell pwsh") {
		t.Fatalf("expected pwsh shell: %v", args)
	}
}
