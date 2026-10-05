package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

const (
	runnerTokenAccountPrefix = "GITLAB_RUNNER_TOKEN_"
	pendingRunnerTokenEnv    = "GITLAB_RUNNER_TOKEN"
	patAccount               = "GITLAB_TOKEN"
)

// KeyringOrDefault returns kr when non-nil, otherwise the process default keyring.
func KeyringOrDefault(kr operatorconfig.Keyring) operatorconfig.Keyring {
	if kr != nil {
		return kr
	}
	return operatorconfig.DefaultKeyring()
}

// RunnerTokenEnv returns the env/account name for a GitLab runner id glrt secret.
func RunnerTokenEnv(runnerID int) (string, error) {
	if runnerID <= 0 {
		return "", fmt.Errorf("config: runner id is required")
	}
	return runnerTokenAccountPrefix + strconv.Itoa(runnerID), nil
}

// RunnerTokenEnvForTag returns the env/account name for a human-readable runner tag glrt secret.
// Example tag homelab-mac → GITLAB_RUNNER_TOKEN_homelab-mac.
func RunnerTokenEnvForTag(tag string) (string, error) {
	tag = operatorconfig.SanitizeSecret(tag)
	if tag == "" {
		return "", fmt.Errorf("config: runner tag is required")
	}
	return runnerTokenAccountPrefix + tag, nil
}

func loadSecretFromEnvOrKeyring(env string, kr operatorconfig.Keyring) (string, error) {
	clean := operatorconfig.SanitizeSecret(os.Getenv(env))
	if clean != "" {
		return clean, nil
	}
	got, err := kr.Get(appName, env)
	if err != nil {
		if errors.Is(err, operatorconfig.ErrNotFound) {
			return "", nil
		}
		return "", errdefs.New("config", errdefs.CodeCreateFailed, "could not load secret from keyring", err)
	}
	return got, nil
}

// StoreRunnerToken persists a glrt for a GitLab runner id in the OS keyring.
func StoreRunnerToken(runnerID int, token string, kr operatorconfig.Keyring) error {
	env, err := RunnerTokenEnv(runnerID)
	if err != nil {
		return errdefs.New("config.StoreRunnerToken", errdefs.CodeCreateFailed, err.Error(), err)
	}
	clean := operatorconfig.SanitizeSecret(token)
	if clean == "" {
		return errdefs.New("config.StoreRunnerToken", errdefs.CodeCreateFailed, "runner token is empty", nil)
	}
	kr = KeyringOrDefault(kr)
	if err := kr.Set(appName, env, clean); err != nil {
		return errdefs.New("config.StoreRunnerToken", errdefs.CodeCreateFailed, "could not store runner token", err)
	}
	return nil
}

// LoadRunnerToken reads glrt for runnerID from env then keyring.
func LoadRunnerToken(runnerID int, kr operatorconfig.Keyring) (string, error) {
	env, err := RunnerTokenEnv(runnerID)
	if err != nil {
		return "", errdefs.New("config.LoadRunnerToken", errdefs.CodeCreateFailed, err.Error(), err)
	}
	return loadSecretFromEnvOrKeyring(env, KeyringOrDefault(kr))
}

// StoreRunnerTokenByTag saves a glrt under GITLAB_RUNNER_TOKEN_<tag> before register.
func StoreRunnerTokenByTag(tag string, token string, kr operatorconfig.Keyring) error {
	env, err := RunnerTokenEnvForTag(tag)
	if err != nil {
		return errdefs.New("config.StoreRunnerTokenByTag", errdefs.CodeCreateFailed, err.Error(), err)
	}
	clean := operatorconfig.SanitizeSecret(token)
	if clean == "" {
		return errdefs.New("config.StoreRunnerTokenByTag", errdefs.CodeCreateFailed, "runner token is empty", nil)
	}
	kr = KeyringOrDefault(kr)
	if err := kr.Set(appName, env, clean); err != nil {
		return errdefs.New("config.StoreRunnerTokenByTag", errdefs.CodeCreateFailed, "could not store runner token", err)
	}
	return nil
}

// LoadRunnerTokenByTag reads glrt for tag from env then keyring.
func LoadRunnerTokenByTag(tag string, kr operatorconfig.Keyring) (string, error) {
	env, err := RunnerTokenEnvForTag(tag)
	if err != nil {
		return "", errdefs.New("config.LoadRunnerTokenByTag", errdefs.CodeCreateFailed, err.Error(), err)
	}
	return loadSecretFromEnvOrKeyring(env, KeyringOrDefault(kr))
}

// StorePendingRunnerToken saves a glrt before GitLab runner id is known.
func StorePendingRunnerToken(token string, kr operatorconfig.Keyring) error {
	clean := operatorconfig.SanitizeSecret(token)
	if clean == "" {
		return errdefs.New("config.StorePendingRunnerToken", errdefs.CodeCreateFailed, "runner token is empty", nil)
	}
	kr = KeyringOrDefault(kr)
	if err := kr.Set(appName, pendingRunnerTokenEnv, clean); err != nil {
		return errdefs.New("config.StorePendingRunnerToken", errdefs.CodeCreateFailed, "could not store runner token", err)
	}
	return nil
}

// LoadPendingRunnerToken reads GITLAB_RUNNER_TOKEN from env then keyring.
func LoadPendingRunnerToken(kr operatorconfig.Keyring) (string, error) {
	return loadSecretFromEnvOrKeyring(pendingRunnerTokenEnv, KeyringOrDefault(kr))
}

// StorePAT saves GITLAB_TOKEN to the keyring and sets process env for this run.
func StorePAT(token string, kr operatorconfig.Keyring) error {
	clean := operatorconfig.SanitizeSecret(token)
	if clean == "" {
		return errdefs.New("config.StorePAT", errdefs.CodeCreateFailed, "PAT is empty", nil)
	}
	kr = KeyringOrDefault(kr)
	if err := kr.Set(appName, patAccount, clean); err != nil {
		return errdefs.New("config.StorePAT", errdefs.CodeCreateFailed, "could not store PAT", err)
	}
	if err := os.Setenv(patAccount, clean); err != nil {
		return errdefs.New("config.StorePAT", errdefs.CodeCreateFailed, "could not set PAT in environment", err)
	}
	return nil
}
