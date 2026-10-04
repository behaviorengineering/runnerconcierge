package inventory

import (
	"runtime"
	"testing"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

func TestClassifyWindowsLocalSystem(t *testing.T) {
	if !service.IsLocalSystemLogon("LocalSystem") {
		t.Fatal("IsLocalSystemLogon")
	}
	_ = errdefs.CodeBadServiceLogon
}

func TestClassifySystemConfig(t *testing.T) {
	var path string
	switch runtime.GOOS {
	case "windows":
		path = "C:\\GitLab-Runner\\config.toml"
	case "darwin":
		path = "/etc/gitlab-runner/config.toml"
	default:
		path = "/etc/gitlab-runner/config.toml"
	}
	if !service.IsSystemConfigPath(path) {
		t.Fatalf("expected system path for %s on %s", path, runtime.GOOS)
	}
	findings := classifyConfigPath(path, true, false)
	if len(findings) == 0 || findings[0].Code != errdefs.CodeSystemConfig {
		t.Fatalf("expected system_config, got %+v", findings)
	}
}
