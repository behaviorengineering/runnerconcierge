package wizard

import (
	"context"
	"fmt"
	"time"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/gitlabrunner"
	"github.com/behaviorengineering/runnerconcierge/pkg/install"
)

const runnerOnlineVerifyTimeout = 2 * time.Minute

// verifyRunnerOnlineOnGitLab polls GitLab until the runner is online or timeout.
func (r *Runner) verifyRunnerOnlineOnGitLab(ctx context.Context, runnerID int) error {
	if r == nil || runnerID <= 0 {
		return nil
	}
	inst := install.New(r.exec)
	if _, err := inst.Ensure(ctx, install.ToolGlab, r.opts.AllowInstall || r.opts.NonInteractive); err != nil {
		return err
	}
	client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)
	if err := client.WaitOnline(ctx, runnerID, runnerOnlineVerifyTimeout); err != nil {
		return errdefs.New("setup", errdefs.CodeOf(err),
			fmt.Sprintf("runner %d is not online on GitLab after %s", runnerID, runnerOnlineVerifyTimeout), err)
	}
	r.out(fmt.Sprintf("setup: runner %d verified online on GitLab", runnerID))
	return nil
}
