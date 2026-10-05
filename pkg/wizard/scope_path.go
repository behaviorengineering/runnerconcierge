package wizard

import (
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

func parentScopePath(opts Options, cp *state.Checkpoint) string {
	if p := strings.TrimSpace(opts.GroupPath); p != "" {
		return p
	}
	if cp != nil {
		if p := strings.TrimSpace(cp.GroupPath); p != "" {
			return p
		}
	}
	if p := strings.TrimSpace(opts.ProjectPath); p != "" {
		return p
	}
	if cp != nil {
		if p := strings.TrimSpace(cp.RepoPath); p != "" {
			return p
		}
	}
	if p := strings.TrimSpace(opts.Preset.RepoPath); p != "" {
		return p
	}
	return ""
}

func runnerIdentityFor(opts Options, cp *state.Checkpoint) (string, error) {
	parent := parentScopePath(opts, cp)
	tag := canonicalTag(opts, cp)
	return RunnerIdentity(parent, tag, hostname())
}
