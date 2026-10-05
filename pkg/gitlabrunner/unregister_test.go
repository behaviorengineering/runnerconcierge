package gitlabrunner

import (
	"slices"
	"strings"
	"testing"
)

func TestBuildUnregisterArgv(t *testing.T) {
	args, err := BuildUnregisterArgv(UnregisterArgs{
		Name:       "my-host",
		ConfigPath: "/cfg.toml",
		URL:        "https://gitlab.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"unregister",
		"--name", "my-host",
		"--config", "/cfg.toml",
		"--url", "https://gitlab.com",
	}
	if !slices.Equal(args, want) {
		t.Fatalf("argv: got %v want %v", args, want)
	}
	if slices.Contains(args, "--non-interactive") {
		t.Fatal("unregister must not pass --non-interactive (gitlab-runner 19.x rejects it)")
	}
}

func TestBuildUnregisterArgv_minimal(t *testing.T) {
	args, err := BuildUnregisterArgv(UnregisterArgs{Name: "runner-a"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(args, []string{"unregister", "--name", "runner-a"}) {
		t.Fatalf("got %v", args)
	}
}

func TestBuildUnregisterArgv_rejectsEmptyName(t *testing.T) {
	_, err := BuildUnregisterArgv(UnregisterArgs{Name: ""})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("err: %v", err)
	}
}
