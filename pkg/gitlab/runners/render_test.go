package runners

import (
	"strings"
	"testing"

	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

func TestFormatInspectBody_serviceOnlyOmitsName(t *testing.T) {
	body := formatInspectBody(&Target{
		Kind:         TargetKindServiceOnly,
		ServiceName:  "com.hector.gitlab-runner-docker-cleanup",
		ServiceKind:  "launchd",
		ServiceState: "running",
		Role:         service.RoleHelper,
		MatchReason:  service.MatchReasonLaunchdLabel,
		UnitPath:     "/Users/me/Library/LaunchAgents/com.hector.gitlab-runner-docker-cleanup.plist",
		Command:      "/bin/bash /cleanup.sh",
		ProcessUp:    true,
	}, "")
	if strings.Contains(body, "Name:") {
		t.Fatalf("should omit name: %q", body)
	}
	if !strings.Contains(body, "Role: helper") {
		t.Fatalf("missing role: %q", body)
	}
	if !strings.Contains(body, "Process up: yes") {
		t.Fatalf("missing process: %q", body)
	}
}

func TestRemoveConfirmMessage_helper(t *testing.T) {
	msg := removeConfirmMessage(&Target{
		Kind:        TargetKindServiceOnly,
		ServiceName: "com.hector.gitlab-runner-docker-cleanup",
		Role:        service.RoleHelper,
	})
	if !strings.Contains(msg, "service unit") {
		t.Fatalf("msg: %q", msg)
	}
}
