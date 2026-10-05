package wizard

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

const (
	glrtPasswordTitle       = "GitLab runner authentication token (glrt). Leave empty to create a new runner."
	glrtResumePasswordTitle = "GitLab runner authentication token (glrt) not found in keyring. Paste from GitLab runner settings."
	patPasswordTitle        = "GitLab personal access token (create_runner scope). Saved to keyring."
)

func (r *Runner) keyring() operatorconfig.Keyring {
	return config.KeyringOrDefault(r.opts.Keyring)
}

func (r *Runner) hasStoredRunnerToken(runnerID int, cp *state.Checkpoint) (string, error) {
	if runnerID > 0 {
		tok, err := config.LoadRunnerToken(runnerID, r.keyring())
		if err != nil || tok != "" {
			return tok, err
		}
	}
	tag := canonicalTag(r.opts, cp)
	if tag != "" {
		tok, err := config.LoadRunnerTokenByTag(tag, r.keyring())
		if err != nil || tok != "" {
			return tok, err
		}
	}
	return config.LoadPendingRunnerToken(r.keyring())
}

func (r *Runner) loadGLRTIntoOpts(runnerID int, cp *state.Checkpoint) error {
	if strings.TrimSpace(r.opts.RunnerToken) != "" {
		return nil
	}
	stored, err := r.hasStoredRunnerToken(runnerID, cp)
	if err != nil {
		return err
	}
	if stored != "" {
		r.opts.RunnerToken = stored
	}
	return nil
}

func (r *Runner) storeRunnerGLRT(runnerID int, tag, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	if tag != "" {
		if err := config.StoreRunnerTokenByTag(tag, token, r.keyring()); err != nil {
			return err
		}
	} else if runnerID == 0 {
		if err := config.StorePendingRunnerToken(token, r.keyring()); err != nil {
			return err
		}
	}
	if runnerID > 0 {
		return config.StoreRunnerToken(runnerID, token, r.keyring())
	}
	return nil
}

// promptCredentials asks for glrt (save-and-forget), then PAT when creating a runner.
func (r *Runner) promptCredentials(ctx context.Context, pr prompt.Prompter, runnerID int, cp *state.Checkpoint) error {
	if r == nil || r.opts.NonInteractive || pr == nil {
		return nil
	}
	if err := r.loadGLRTIntoOpts(runnerID, cp); err != nil {
		return err
	}
	if strings.TrimSpace(r.opts.RunnerToken) != "" {
		return nil
	}

	tag := canonicalTag(r.opts, cp)
	val, err := pr.Password(ctx, glrtPasswordTitle)
	if err != nil {
		return err
	}
	val = strings.TrimSpace(val)
	if val != "" {
		r.opts.RunnerToken = val
		if err := r.storeRunnerGLRT(runnerID, tag, val); err != nil {
			return err
		}
	}

	if strings.TrimSpace(r.opts.RunnerToken) != "" || strings.TrimSpace(r.opts.PAT) != "" {
		return nil
	}

	opts := config.Options("")
	opts.Secrets = []operatorconfig.Secret{{Env: "GITLAB_TOKEN", Required: false}}
	if err := operatorconfig.ResolveSecrets(opts, r.keyring()); err != nil {
		return err
	}
	if strings.TrimSpace(os.Getenv("GITLAB_TOKEN")) != "" {
		return nil
	}

	pat, err := pr.Password(ctx, patPasswordTitle)
	if err != nil {
		return err
	}
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return nil
	}
	r.opts.PAT = pat
	return config.StorePAT(pat, r.keyring())
}

// ensureGLRTInteractive reprompts for glrt when checkpoint has runner_id but keyring miss.
func (r *Runner) ensureGLRTInteractive(ctx context.Context, pr prompt.Prompter, cp *state.Checkpoint) error {
	if r == nil || r.opts.NonInteractive || pr == nil || cp == nil {
		return nil
	}
	if err := r.loadGLRTIntoOpts(cp.RunnerID, cp); err != nil {
		return err
	}
	if strings.TrimSpace(r.opts.RunnerToken) != "" {
		return nil
	}
	if cp.RunnerID <= 0 {
		return nil
	}

	val, err := pr.Password(ctx, glrtResumePasswordTitle)
	if err != nil {
		return err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return errdefs.New("setup", errdefs.CodeInvalidScope,
			fmt.Sprintf("runner %d exists but glrt token missing; pass --token or recreate in GitLab", cp.RunnerID),
			nil)
	}
	r.opts.RunnerToken = val
	tag := canonicalTag(r.opts, cp)
	return r.storeRunnerGLRT(cp.RunnerID, tag, val)
}
