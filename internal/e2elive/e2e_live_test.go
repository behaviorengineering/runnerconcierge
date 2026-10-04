//go:build e2e_live && (darwin || windows)

package e2elive

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/inventory"
)

func TestObserveStatus(t *testing.T) {
	mode := os.Getenv("E2E_LIVE_MODE")
	if mode == "repair-prod" && os.Getenv("E2E_LIVE_ALLOW_PROD") != "1" {
		t.Skip("repair-prod requires E2E_LIVE_ALLOW_PROD=1")
	}
	if _, err := gitexec.New().LookPath("gitlab-runner"); err != nil {
		t.Skip("gitlab-runner not on PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	rep, err := inventory.Run(ctx, inventory.Options{Exec: gitexec.New()})
	if err != nil {
		t.Fatal(err)
	}
	if rep.GOOS == "" {
		t.Fatal("expected GOOS")
	}
}
