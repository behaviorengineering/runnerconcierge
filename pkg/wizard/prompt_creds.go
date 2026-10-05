package wizard

import (
	"context"
	"strings"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/gitlabrunner"
	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

const glabLoginConfirmTitle = "GitLab CLI is not logged in. Log in with glab now?"

func (r *Runner) keyring() operatorconfig.Keyring {
	return config.KeyringOrDefault(r.opts.Keyring)
}

func (r *Runner) hasStoredRunnerToken(cp *state.Checkpoint) (string, error) {
	return loadStoredRunnerGLRT(r.opts, cp, r.keyring())
}

func (r *Runner) loadGLRTIntoOpts(runnerID int, cp *state.Checkpoint) error {
	if strings.TrimSpace(r.opts.RunnerToken) != "" {
		return nil
	}
	stored, err := r.hasStoredRunnerToken(cp)
	if err != nil {
		return err
	}
	if stored != "" {
		r.opts.RunnerToken = stored
	}
	return nil
}

func (r *Runner) storeRunnerGLRT(cp *state.Checkpoint, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	identity, idErr := runnerIdentityFor(r.opts, cp)
	return storeRunnerGLRTKeyring(identity, idErr, token, r.keyring())
}

func (r *Runner) hasRunnerToken(cp *state.Checkpoint) bool {
	return strings.TrimSpace(r.opts.RunnerToken) != ""
}

// hydrateRunnerToken loads --token / keyring glrt into opts without prompting.
func (r *Runner) hydrateRunnerToken(cp *state.Checkpoint) error {
	if r == nil || cp == nil {
		return nil
	}
	runnerID := cp.RunnerID
	return r.loadGLRTIntoOpts(runnerID, cp)
}

// ensureLoggedIn requires a glab session when no runner token is already present.
func (r *Runner) ensureLoggedIn(ctx context.Context, pr prompt.Prompter) (gitlabrunner.GitLabUser, error) {
	if r == nil {
		return gitlabrunner.GitLabUser{}, errdefs.New("setup", errdefs.CodeAuthRequired, "wizard is nil", nil)
	}
	client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)
	user, err := client.WhoAmI(ctx)
	if err == nil && user.ID > 0 {
		return user, nil
	}
	if r.opts.NonInteractive {
		return gitlabrunner.GitLabUser{}, errdefs.New("setup", errdefs.CodeAuthRequired,
			"run glab auth login or pass --token", err)
	}
	if pr == nil {
		return gitlabrunner.GitLabUser{}, errdefs.New("setup", errdefs.CodeAuthRequired,
			"run glab auth login or pass --token", err)
	}
	ok, err := pr.Confirm(ctx, glabLoginConfirmTitle)
	if err != nil {
		return gitlabrunner.GitLabUser{}, err
	}
	if !ok {
		return gitlabrunner.GitLabUser{}, errdefs.New("setup", errdefs.CodeAuthRequired,
			"run glab auth login or pass --token", nil)
	}
	if err := client.AuthLogin(ctx); err != nil {
		return gitlabrunner.GitLabUser{}, errdefs.New("setup", errdefs.CodeAuthRequired,
			"run glab auth login or pass --token", err)
	}
	user, err = client.WhoAmI(ctx)
	if err != nil || user.ID <= 0 {
		return gitlabrunner.GitLabUser{}, errdefs.New("setup", errdefs.CodeAuthRequired,
			"run glab auth login or pass --token", err)
	}
	return user, nil
}

func (r *Runner) ensureRunnerAccess(ctx context.Context, user gitlabrunner.GitLabUser, cp *state.Checkpoint) error {
	if r == nil || cp == nil {
		return nil
	}
	if r.hasRunnerToken(cp) {
		return nil
	}
	runnerType := r.effectiveRunnerType()
	if runnerType == "" {
		return nil
	}
	var path string
	switch runnerType {
	case "group_type":
		path = strings.TrimSpace(r.opts.GroupPath)
	case "project_type":
		path = strings.TrimSpace(r.opts.ProjectPath)
		if path == "" {
			path = strings.TrimSpace(r.preset.RepoPath)
		}
	default:
		return nil
	}
	client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)
	return client.EnsureCanCreateRunner(ctx, runnerType, path, user)
}

// resetRunnerGLRTIfNeeded mints glrt via GitLab API when resuming with runner_id and empty keyring.
func (r *Runner) resetRunnerGLRTIfNeeded(ctx context.Context, cp *state.Checkpoint) error {
	if r == nil || cp == nil {
		return nil
	}
	if r.hasRunnerToken(cp) {
		return nil
	}
	if cp.RunnerID <= 0 {
		return nil
	}
	client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)
	tok, err := client.ResetAuthenticationToken(ctx, cp.RunnerID)
	if err != nil {
		return errdefs.New("setup", errdefs.CodeOf(err), "could not reset runner authentication token", err)
	}
	r.opts.RunnerToken = tok
	return r.storeRunnerGLRT(cp, tok)
}
