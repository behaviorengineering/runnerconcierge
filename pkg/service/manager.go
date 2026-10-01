package service

import (
	"context"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
)

// InstallOpts configures service installation.
type InstallOpts struct {
	BinaryPath       string
	ConfigPath       string
	WorkingDirectory string
	WindowsUser      string
	WindowsPassword  string // memory only; never persisted
	UseBrewServices  bool
}

// Manager installs and starts the runner service.
type Manager interface {
	Install(ctx context.Context, opts InstallOpts) error
	Start(ctx context.Context) error
	Status(ctx context.Context) (string, error)
}

// New returns a platform manager.
func New(exec gitexec.Exec) Manager {
	if exec == nil {
		panic("service: exec is nil")
	}
	return newPlatformManager(exec)
}
