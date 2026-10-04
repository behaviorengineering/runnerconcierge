package cli

import (
	"context"
	"io"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	gitlabrunners "github.com/behaviorengineering/runnerconcierge/pkg/gitlab/runners"
	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
)

func runGitLabRunners(ctx context.Context, stdout, stderr io.Writer, cfg gitlabrunners.Config) error {
	cfg.Exec = gitexec.New()
	cfg.Out = stdout
	cfg.ErrOut = stderr
	if cfg.Prompter == nil && !cfg.NonInteractive {
		cfg.Prompter = prompt.ForTTY()
	}
	ctrl, err := cfg.Create()
	if err != nil {
		return err
	}
	ctx, cancel := withDeadline(ctx, 10*time.Minute)
	defer cancel()
	return ctrl.Run(ctx)
}
