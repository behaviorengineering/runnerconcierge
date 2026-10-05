package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

const runnerTagInputTitle = "Runner tag (CI jobs match this; glrt keyring entry is GITLAB_RUNNER_TOKEN_<tag>)"

// canonicalTag returns the primary tag used for CI and keyring account names.
func canonicalTag(opts Options, cp *state.Checkpoint) string {
	if len(opts.TagList) > 0 {
		if t := strings.TrimSpace(opts.TagList[0]); t != "" {
			return t
		}
	}
	if cp != nil && len(cp.Tags) > 0 {
		if t := strings.TrimSpace(cp.Tags[0]); t != "" {
			return t
		}
	}
	return ""
}

func (r *Runner) promptCanonicalTag(ctx context.Context, pr prompt.Prompter, cp *state.Checkpoint) error {
	if r == nil {
		return fmt.Errorf("wizard: runner is nil")
	}
	if canonicalTag(r.opts, cp) != "" {
		return nil
	}
	if r.opts.NonInteractive || pr == nil {
		return nil
	}
	defaultVal := ""
	if cp != nil && len(cp.Tags) > 0 {
		defaultVal = strings.TrimSpace(cp.Tags[0])
	}
	val, err := pr.Input(ctx, runnerTagInputTitle, "e.g. homelab-mac", defaultVal)
	if err != nil {
		return err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return fmt.Errorf("wizard: runner tag is required")
	}
	r.opts.TagList = []string{val}
	if cp != nil {
		cp.Tags = []string{val}
		if err := r.saveCheckpoint(cp); err != nil {
			return err
		}
	}
	return nil
}
