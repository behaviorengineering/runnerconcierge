package service

import (
	"fmt"
	"strings"
	"testing"
)

func TestGitLabServiceNotInstalled(t *testing.T) {
	err := fmt.Errorf("gitlab-runner status: Runtime platform arch=arm64: gitlab-runner: the service is not installed: exit status 1")
	if !gitlabServiceNotInstalled(err) {
		t.Fatal("expected not-installed detection")
	}
	if gitlabServiceNotInstalled(nil) {
		t.Fatal("nil")
	}
}

func TestNativeStatusLineSkipsRuntimeBanner(t *testing.T) {
	out := []byte("Runtime platform                                    arch=arm64 os=darwin\ngitlab-runner: Service is running\n")
	got := nativeStatusLine(out)
	if got != "gitlab-runner: Service is running" {
		t.Fatalf("got %q", got)
	}
}

func TestBrewGitLabRunnerStatusLine(t *testing.T) {
	list := "Name          Status User   File\ncloudflared   none\ngitlab-runner started         hector ~/Library/LaunchAgents/sh.brew.gitlab-runner.plist\n"
	got := brewGitLabRunnerStatusLine(list)
	if got != "gitlab-runner brew_services started user=hector" {
		t.Fatalf("got %q", got)
	}
	if brewGitLabRunnerStatusLine("unbound none") != "" {
		t.Fatal("expected empty when gitlab-runner absent")
	}
}

func TestStripANSI(t *testing.T) {
	raw := "Runtime platform                                  \x1b[0;m  arch\x1b[0;m=arm64"
	if strings.Contains(stripANSI(raw), "\x1b") {
		t.Fatal("ansi remained")
	}
}
