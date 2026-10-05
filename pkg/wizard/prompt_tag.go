package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

const runnerNameInputTitle = "Runner name (short label for this machine)"

// canonicalTag returns the runner name label from flags or checkpoint (also the primary GitLab job tag).
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

func (r *Runner) promptRunnerName(ctx context.Context, pr prompt.Prompter, cp *state.Checkpoint) error {
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
	val, err := pr.Input(ctx, runnerNameInputTitle, "e.g. macos-dss", defaultVal)
	if err != nil {
		return err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return fmt.Errorf("wizard: runner name is required")
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
