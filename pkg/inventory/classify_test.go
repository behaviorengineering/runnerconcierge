package inventory

import (
	"testing"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

func TestClassifyWindowsLocalSystem(t *testing.T) {
	if service.IsLocalSystemLogon("LocalSystem") != true {
		t.Fatal("IsLocalSystemLogon")
	}
	_ = errdefs.CodeBadServiceLogon
}

func TestClassifySystemConfig(t *testing.T) {
	findings := classifyConfigPath("/etc/gitlab-runner/config.toml", true, false)
	if len(findings) == 0 || findings[0].Code != errdefs.CodeSystemConfig {
		t.Fatalf("expected system_config, got %+v", findings)
	}
}
