package wizard

import (
	"context"
	"errors"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/gitlabrunner"
	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

func invalidRegisterToken(err error) bool {
	for e := err; e != nil; e = errors.Unwrap(e) {
		if strings.Contains(strings.ToLower(e.Error()), "is not valid") {
			return true
		}
	}
	return false
}

func (r *Runner) registerWithRetry(ctx context.Context, pr prompt.Prompter, runnerBin string, cp *state.Checkpoint, token, cfgPath, workDir string) error {
	if r == nil {
		return errdefs.New("setup", errdefs.CodeRegisterFailed, "wizard is nil", nil)
	}
	err := r.registerOnce(ctx, runnerBin, cp, token, cfgPath, workDir)
	if err == nil || !invalidRegisterToken(err) {
		return err
	}
	if r.out != nil {
		r.out("setup: GitLab rejected stored runner token; minting a new one")
	}
	if refreshErr := r.refreshGLRTAfterInvalidRegister(ctx, pr, cp); refreshErr != nil {
		return refreshErr
	}
	return r.registerOnce(ctx, runnerBin, cp, r.opts.RunnerToken, cfgPath, workDir)
}

func (r *Runner) registerOnce(ctx context.Context, runnerBin string, cp *state.Checkpoint, token, cfgPath, workDir string) error {
	regArgs, err := gitlabrunner.BuildRegisterArgv(gitlabrunner.RegisterArgs{
		URL:              r.gitlabURL(),
		Token:            token,
		Name:             r.runnerName(cp),
		Executor:         r.executor(),
		ConfigPath:       cfgPath,
		WorkingDirectory: workDir,
		DockerImage:      r.dockerImage(),
	})
	if err != nil {
		return errdefs.New("setup", errdefs.CodeRegisterFailed, "could not build register argv", err)
	}
	client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)
	if err := client.Register(ctx, runnerBin, regArgs); err != nil {
		return err
	}
	return nil
}

func (r *Runner) refreshGLRTAfterInvalidRegister(ctx context.Context, pr prompt.Prompter, cp *state.Checkpoint) error {
	if r == nil || cp == nil {
		return errdefs.New("setup", errdefs.CodeCreateFailed, "wizard or checkpoint is nil", nil)
	}
	r.opts.RunnerToken = ""
	r.opts.IgnoreKeyringGLRT = true
	if cp.RunnerID > 0 {
		if err := r.resetRunnerGLRTIfNeeded(ctx, cp); err != nil {
			if r.out != nil {
				r.out("setup: could not reset runner token: " + err.Error())
			}
			cp.RunnerID = 0
		} else if r.hasRunnerToken(cp) {
			return nil
		}
	}
	user, err := r.ensureLoggedIn(ctx, pr)
	if err != nil {
		return err
	}
	if err := r.ensureRunnerAccess(ctx, user, cp); err != nil {
		return err
	}
	_, _, err = r.resolveRegisterToken(ctx, cp)
	return err
}
