package inventory

import (
	"testing"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

func TestHasBlockingForTarget(t *testing.T) {
	rep := &Report{
		Findings: []Finding{
			{Code: errdefs.CodeServiceLogon, Block: true, ConfigPath: "/other/config.toml"},
			{Code: errdefs.CodeServiceLogon, Block: true, ConfigPath: "/fixture/config.toml"},
		},
	}
	if !HasBlockingForTarget(rep, "/fixture/config.toml", "") {
		t.Fatal("expected blocking for fixture config")
	}
	if HasBlockingForTarget(rep, "/mine.toml", "") {
		t.Fatal("other config should not block target scope")
	}
	byService := &Report{Findings: []Finding{{Block: true, Service: "runnerconcierge-e2e-abc"}}}
	if !HasBlockingForTarget(byService, "", "runnerconcierge-e2e-abc") {
		t.Fatal("expected blocking by service name")
	}
}
