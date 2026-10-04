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
	ServiceName      string
	WindowsUser      string
	WindowsPassword  string // memory only; never persisted
	UseBrewServices  bool
}

// StartOpts configures service start.
type StartOpts struct {
	ServiceName string
	UseBrew     bool
}

// StopOpts configures service stop.
type StopOpts struct {
	ServiceName string
	UseBrew     bool
}

// Ownership describes a discovered runner service unit.
type Ownership struct {
	ServiceName string
	State       string
	LogonUser   string
	ConfigPath  string
	Kind        string
}

// UninstallOpts configures service removal.
type UninstallOpts struct {
	BinaryPath  string
	ConfigPath  string
	ServiceName string
	UseBrew     bool
}

// Manager installs and starts the runner service.
type Manager interface {
	Install(ctx context.Context, opts InstallOpts) error
	Start(ctx context.Context, opts StartOpts) error
	Stop(ctx context.Context, opts StopOpts) error
	Status(ctx context.Context) (string, error)
	Uninstall(ctx context.Context, opts UninstallOpts) error
	ListOwnership(ctx context.Context) ([]Ownership, error)
}

// New returns a platform manager.
func New(exec gitexec.Exec) Manager {
	if exec == nil {
		panic("service: exec is nil")
	}
	return newPlatformManager(exec)
}
