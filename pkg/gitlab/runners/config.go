package runners

import (
	"fmt"
	"io"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/gitlabrunner"
	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

// Config builds a GitLab runners controller.
type Config struct {
	Exec            gitexec.Exec
	Out             io.Writer
	ErrOut          io.Writer
	Prompter        prompt.Prompter
	Service         service.Manager
	Client          *gitlabrunner.Client
	GitLabURL       string
	PAT             string
	NonInteractive  bool
	AllowYes        bool
	Name            string
	ServiceName     string
	ConfigPath      string
	GitLabID        int
	Action          Action
	JSON            bool
	LocalOnly       bool
	WindowsPassword string
}

// Controller runs the GitLab runners control plane.
type Controller struct {
	cfg Config
}

// Create builds a Controller from Config.
func (cfg Config) Create() (*Controller, error) {
	if cfg.Exec == nil {
		panic("runners: exec is nil")
	}
	if cfg.Out == nil {
		return nil, fmt.Errorf("runners: Out writer is required")
	}
	if cfg.Service == nil {
		cfg.Service = service.New(cfg.Exec)
	}
	if cfg.Client == nil {
		base := cfg.GitLabURL
		if base == "" {
			userCfg, err := config.Load("")
			if err == nil && userCfg != nil {
				base = userCfg.GitLabURL
			}
		}
		if base == "" {
			base = "https://gitlab.com"
		}
		cfg.Client = gitlabrunner.NewClient(base, cfg.Exec, nil)
	}
	return &Controller{cfg: cfg}, nil
}
