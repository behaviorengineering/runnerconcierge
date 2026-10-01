package concierge

import (
	"context"

	"github.com/behaviorengineering/runnerconcierge/pkg/preset"
	"github.com/behaviorengineering/runnerconcierge/pkg/wizard"
)

// Options mirrors wizard.Options for host presets.
type Options = wizard.Options

// Run executes setup with a host preset.
func Run(ctx context.Context, pre preset.Preset, opts Options) error {
	opts.Preset = pre
	run, err := wizard.New(opts, nil)
	if err != nil {
		return err
	}
	return run.Run(ctx)
}
