package cli

import (
	"context"
	"io"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/cleanup"
)

func runCleanup(ctx context.Context, stdout, stderr io.Writer, cfg cleanup.Config) error {
	cfg.Exec = gitexec.New()
	if cfg.Out == nil {
		cfg.Out = stdout
	}
	if cfg.ErrOut == nil {
		cfg.ErrOut = stderr
	}
	ctrl, err := cfg.Create()
	if err != nil {
		return err
	}
	ctx, cancel := withDeadline(ctx, 10*time.Minute)
	defer cancel()
	return ctrl.Run(ctx)
}

func runCleanupInstall(ctx context.Context, stdout, stderr io.Writer, cfg cleanup.Config) error {
	cfg.Exec = gitexec.New()
	if cfg.Out == nil {
		cfg.Out = stdout
	}
	if cfg.ErrOut == nil {
		cfg.ErrOut = stderr
	}
	ctrl, err := cfg.Create()
	if err != nil {
		return err
	}
	ctx, cancel := withDeadline(ctx, 10*time.Minute)
	defer cancel()
	return ctrl.Install(ctx)
}

func runCleanupUninstall(ctx context.Context, stdout, stderr io.Writer, cfg cleanup.Config) error {
	cfg.Exec = gitexec.New()
	if cfg.Out == nil {
		cfg.Out = stdout
	}
	if cfg.ErrOut == nil {
		cfg.ErrOut = stderr
	}
	ctrl, err := cfg.Create()
	if err != nil {
		return err
	}
	ctx, cancel := withDeadline(ctx, 10*time.Minute)
	defer cancel()
	return ctrl.Uninstall(ctx)
}
