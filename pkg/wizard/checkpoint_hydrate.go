package wizard

import (
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

// applyCheckpointToOpts copies non-secret checkpoint fields into empty Options (CLI flags win).
func applyCheckpointToOpts(opts *Options, cp *state.Checkpoint) {
	if opts == nil || cp == nil {
		return
	}
	if len(opts.TagList) == 0 && len(cp.Tags) > 0 {
		opts.TagList = append([]string(nil), cp.Tags...)
	}
	if strings.TrimSpace(opts.Executor) == "" && strings.TrimSpace(cp.Executor) != "" {
		opts.Executor = strings.TrimSpace(cp.Executor)
	}
	if strings.TrimSpace(opts.ProjectPath) == "" && strings.TrimSpace(cp.RepoPath) != "" {
		opts.ProjectPath = strings.TrimSpace(cp.RepoPath)
	}
	if strings.TrimSpace(opts.GroupPath) == "" && strings.TrimSpace(cp.GroupPath) != "" {
		opts.GroupPath = strings.TrimSpace(cp.GroupPath)
	}
	if strings.TrimSpace(opts.RunnerType) == "" {
		if strings.TrimSpace(opts.GroupPath) != "" {
			opts.RunnerType = runnerTypeGroup
		} else if strings.TrimSpace(opts.ProjectPath) != "" {
			opts.RunnerType = runnerTypeProject
		}
	}
}
