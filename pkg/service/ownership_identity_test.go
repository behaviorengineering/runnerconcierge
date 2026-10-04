package service

import "testing"

func TestClassifyRole(t *testing.T) {
	cases := []struct {
		name, kind, cmd, want string
	}{
		{"runnerconcierge-e2e-abc", "launchd", "", RoleFixture},
		{"gitlab-runner", "brew_services", "", RoleSupervisor},
		{"gitlab-runner", "launchd", "/usr/local/bin/gitlab-runner run --config /c.toml", RoleSupervisor},
		{"com.hector.gitlab-runner-docker-cleanup", "launchd", "/bin/bash /cleanup.sh", RoleHelper},
		{"com.hector.gitlab-runner-docker-cleanup", "launchd", "", RoleHelper},
		{"runnerconcierge-docker-cleanup", "launchd", "/usr/bin/runnerconcierge cleanup", RoleHelper},
		{"other", "launchd", "", RoleUnknown},
	}
	for _, tc := range cases {
		got := ClassifyRole(tc.name, tc.kind, tc.cmd)
		if got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestIsRunnerServiceName(t *testing.T) {
	if !IsRunnerServiceName("com.hector.gitlab-runner-docker-cleanup") {
		t.Fatal("expected match")
	}
	if IsRunnerServiceName("nginx") {
		t.Fatal("unexpected match")
	}
}
