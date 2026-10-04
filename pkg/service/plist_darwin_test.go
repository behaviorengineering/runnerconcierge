//go:build darwin

package service

import "testing"

func TestParseLaunchdPlist_supervisor(t *testing.T) {
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist><dict>
<key>ProgramArguments</key>
<array>
<string>/usr/local/bin/gitlab-runner</string>
<string>run</string>
<string>--config</string>
<string>/Users/me/.gitlab-runner/config.toml</string>
</array>
</dict></plist>`
	info := parseLaunchdPlist([]byte(plist))
	if len(info.Argv) < 2 {
		t.Fatalf("argv: %v", info.Argv)
	}
	if info.ConfigPath != "/Users/me/.gitlab-runner/config.toml" {
		t.Fatalf("config: %q", info.ConfigPath)
	}
	role := ClassifyRole("gitlab-runner", "launchd", joinArgv(info.Argv))
	if role != RoleSupervisor {
		t.Fatalf("role %q", role)
	}
}

func TestParseLaunchdPlist_helper(t *testing.T) {
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist><dict>
<key>ProgramArguments</key>
<array>
<string>/bin/bash</string>
<string>/cleanup.sh</string>
</array>
</dict></plist>`
	info := parseLaunchdPlist([]byte(plist))
	role := ClassifyRole("com.hector.gitlab-runner-docker-cleanup", "launchd", joinArgv(info.Argv))
	if role != RoleHelper {
		t.Fatalf("role %q", role)
	}
}
