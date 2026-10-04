//go:build windows

package service

import (
	"context"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/cleanup"
)

func windowsCleanupTaskOwnership(ctx context.Context, exec gitexec.Exec) *Ownership {
	_, err := exec.Run(ctx, "schtasks", "/Query", "/TN", cleanup.UnitName)
	if err != nil {
		return nil
	}
	return &Ownership{
		ServiceName: cleanup.UnitName,
		State:       "ready",
		Kind:        "scheduled_task",
		UnitPath:    cleanup.UnitName,
		Command:     SanitizeCommand("runnerconcierge cleanup"),
		MatchReason: MatchReasonRunnerconciergeCleanup,
		Role:        RoleHelper,
		ProcessUp:   false,
	}
}
