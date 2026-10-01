package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
)

//go:embed config.yaml.example
var exampleYAML []byte

const appName = "runnerconcierge"

// UserConfig is the operator-facing YAML shape.
type UserConfig struct {
	GitLabURL         string `yaml:"gitlab_url"`
	DefaultExecutor   string `yaml:"default_executor"`
	DescriptionPrefix string `yaml:"description_prefix"`
}

// Options returns operatorconfig discovery options.
func Options(configPath string) operatorconfig.Options {
	return operatorconfig.Options{
		App:            appName,
		ConfigEnv:      "RUNNERCONCIERGE_CONFIG",
		ConfigFlagPath: configPath,
		EnvPrefix:      "RUNNERCONCIERGE",
	}
}

// Init writes example config when missing.
func Init(force bool) (string, error) {
	written, err := operatorconfig.InitUserConfig(Options(""), exampleYAML, force)
	if err != nil {
		return "", err
	}
	path, err := operatorconfig.UserConfigPath(appName, "config.yaml")
	if err != nil {
		return "", err
	}
	if !written {
		return path, nil
	}
	return path, nil
}

// Load reads user config via operatorconfig.
func Load(configPath string) (*UserConfig, error) {
	res, err := operatorconfig.Load(Options(configPath))
	if err != nil {
		return nil, err
	}
	cfg := &UserConfig{
		GitLabURL:       "https://gitlab.com",
		DefaultExecutor: "shell",
	}
	if res.Viper == nil {
		return cfg, nil
	}
	if v := res.Viper.GetString("gitlab_url"); v != "" {
		cfg.GitLabURL = v
	}
	if v := res.Viper.GetString("default_executor"); v != "" {
		cfg.DefaultExecutor = v
	}
	if v := res.Viper.GetString("description_prefix"); v != "" {
		cfg.DescriptionPrefix = v
	}
	return cfg, nil
}

// ConfigDir returns the user config directory.
func ConfigDir() (string, error) {
	path, err := operatorconfig.UserConfigPath(appName, "config.yaml")
	if err != nil {
		return "", err
	}
	return filepath.Dir(path), nil
}

// ResolvePAT returns GITLAB_TOKEN from env or keyring via operatorconfig secrets list.
func ResolvePAT() (string, error) {
	opts := Options("")
	opts.Secrets = []operatorconfig.Secret{{Env: "GITLAB_TOKEN", Required: false}}
	if err := operatorconfig.ResolveSecrets(opts, nil); err != nil {
		return "", err
	}
	val := os.Getenv("GITLAB_TOKEN")
	if val == "" {
		return "", fmt.Errorf("config: GITLAB_TOKEN not set; run glab auth login or set secret")
	}
	return val, nil
}
