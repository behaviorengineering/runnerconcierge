package wizard

import (
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

func (r *Runner) saveCheckpoint(cp *state.Checkpoint) error {
	if r == nil || r.store == nil {
		return nil
	}
	if err := r.store.Save(cp); err != nil {
		return errdefs.New("setup", errdefs.CodeProcessConflict, "could not save checkpoint", err)
	}
	return nil
}

func (r *Runner) syncSetupCheckpoint(cp *state.Checkpoint) {
	if cp == nil {
		return
	}
	if len(r.opts.TagList) > 0 {
		cp.Tags = append([]string(nil), r.opts.TagList...)
	}
	if e := strings.TrimSpace(r.opts.Executor); e != "" {
		cp.Executor = e
	}
	if p := r.opts.ProjectPath; p != "" {
		cp.RepoPath = p
	}
	if g := r.opts.GroupPath; g != "" {
		cp.GroupPath = g
	}
}
