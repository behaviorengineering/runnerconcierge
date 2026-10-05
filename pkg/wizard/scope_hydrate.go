package wizard

import (
	"context"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/gitlabrunner"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

func (r *Runner) needsScopeHydration(cp *state.Checkpoint) bool {
	if r == nil || cp == nil || cp.RunnerID <= 0 {
		return false
	}
	if strings.TrimSpace(r.opts.GroupPath) != "" || r.preset.GroupID > 0 {
		return false
	}
	if strings.TrimSpace(r.opts.ProjectPath) != "" || strings.TrimSpace(r.preset.RepoPath) != "" {
		return false
	}
	if strings.TrimSpace(cp.GroupPath) != "" || strings.TrimSpace(cp.RepoPath) != "" {
		return false
	}
	return true
}

func (r *Runner) hydrateRunnerScopeFromGitLab(ctx context.Context, cp *state.Checkpoint) error {
	if r == nil || cp == nil || cp.RunnerID <= 0 {
		return nil
	}
	client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)
	runnerType, path, err := client.ResolveRunnerScope(ctx, cp.RunnerID)
	if err != nil {
		return errdefs.New("setup", errdefs.CodeOf(err), "could not load runner scope from GitLab", err)
	}
	switch runnerType {
	case "group_type":
		r.opts.RunnerType = runnerTypeGroup
		r.opts.GroupPath = path
	case "project_type":
		r.opts.RunnerType = runnerTypeProject
		r.opts.ProjectPath = path
	default:
		return errdefs.New("setup", errdefs.CodeInvalidScope, "unsupported runner scope "+runnerType, nil)
	}
	r.syncSetupCheckpoint(cp)
	return r.saveCheckpoint(cp)
}
