package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/gitlabrunner"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

// createRunnerFunc creates a GitLab runner and returns id + glrt.
type createRunnerFunc func(ctx context.Context) (int, string, error)

// resolveRegisterToken loads or creates a glrt and persists it to the keyring.
func resolveRegisterToken(
	ctx context.Context,
	opts Options,
	cp *state.Checkpoint,
	kr operatorconfig.Keyring,
	create createRunnerFunc,
) (token string, runnerID int, err error) {
	if cp == nil {
		return "", 0, fmt.Errorf("wizard: checkpoint is nil")
	}
	kr = config.KeyringOrDefault(kr)
	token = strings.TrimSpace(opts.RunnerToken)
	runnerID = cp.RunnerID

	tag := canonicalTag(opts, cp)

	if token == "" && runnerID > 0 {
		token, err = config.LoadRunnerToken(runnerID, kr)
		if err != nil {
			return "", runnerID, err
		}
	}
	if token == "" && tag != "" {
		token, err = config.LoadRunnerTokenByTag(tag, kr)
		if err != nil {
			return "", runnerID, err
		}
	}
	if token == "" {
		token, err = config.LoadPendingRunnerToken(kr)
		if err != nil {
			return "", runnerID, err
		}
	}

	if token == "" && runnerID == 0 {
		if create == nil {
			return "", 0, errdefs.New("setup", errdefs.CodeCreateFailed, "runner token is required", nil)
		}
		runnerID, token, err = create(ctx)
		if err != nil {
			return "", runnerID, err
		}
		cp.RunnerID = runnerID
		if err := storeRunnerGLRTKeyring(runnerID, tag, token, kr); err != nil {
			return "", runnerID, err
		}
	} else if token == "" && runnerID > 0 {
		return "", runnerID, errdefs.New("setup", errdefs.CodeInvalidScope,
			fmt.Sprintf("runner %d exists but glrt token missing; pass --token or recreate in GitLab", runnerID),
			nil)
	} else if token != "" {
		if err := storeRunnerGLRTKeyring(runnerID, tag, token, kr); err != nil {
			return "", runnerID, err
		}
	}

	if strings.TrimSpace(token) == "" {
		return "", runnerID, errdefs.New("setup", errdefs.CodeCreateFailed, "runner token is required", nil)
	}
	return token, runnerID, nil
}

func storeRunnerGLRTKeyring(runnerID int, tag, token string, kr operatorconfig.Keyring) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	if tag != "" {
		if err := config.StoreRunnerTokenByTag(tag, token, kr); err != nil {
			return err
		}
	} else if runnerID == 0 {
		if err := config.StorePendingRunnerToken(token, kr); err != nil {
			return err
		}
	}
	if runnerID > 0 {
		return config.StoreRunnerToken(runnerID, token, kr)
	}
	return nil
}

func (r *Runner) resolveRegisterToken(ctx context.Context, cp *state.Checkpoint) (token string, runnerID int, err error) {
	create := func(ctx context.Context) (int, string, error) {
		client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)
		req, err := r.createRequest(ctx)
		if err != nil {
			return 0, "", err
		}
		runnerID, token, err := client.CreateRunner(ctx, req, "")
		if err != nil {
			return 0, "", errdefs.New("setup", errdefs.CodeOf(err), "could not create GitLab runner", err)
		}
		cp.RunnerID = runnerID
		cp.Description = req.Description
		cp.Tags = req.TagList
		cp.Executor = r.executor()
		if err := r.store.Save(cp); err != nil {
			return 0, "", errdefs.New("setup", errdefs.CodeProcessConflict, "could not save checkpoint", err)
		}
		return runnerID, token, nil
	}
	token, runnerID, err = resolveRegisterToken(ctx, r.opts, cp, r.keyring(), create)
	if err != nil {
		return "", runnerID, err
	}
	r.opts.RunnerToken = token
	cp.RunnerID = runnerID
	return token, runnerID, nil
}
