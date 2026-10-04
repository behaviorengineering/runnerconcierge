//go:build !darwin && !windows

package service

import (
	"context"
	"fmt"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

type stubManager struct{}

func newPlatformManager(exec gitexec.Exec) Manager {
	return &stubManager{}
}

func (s *stubManager) Install(ctx context.Context, opts InstallOpts) error {
	return errdefs.New("service.Install", errdefs.CodeUnsupportedOS, "service install supported on darwin and windows only", nil)
}

func (s *stubManager) Start(ctx context.Context) error {
	return fmt.Errorf("service: unsupported platform")
}

func (s *stubManager) Status(ctx context.Context) (string, error) {
	return "", fmt.Errorf("service: unsupported platform")
}

func (s *stubManager) Uninstall(ctx context.Context, opts UninstallOpts) error {
	return errdefs.New("service.Uninstall", errdefs.CodeUnsupportedOS, "service uninstall supported on darwin and windows only", nil)
}

func (s *stubManager) ListOwnership(ctx context.Context) ([]Ownership, error) {
	return nil, errdefs.New("service.ListOwnership", errdefs.CodeUnsupportedOS, "service ownership supported on darwin and windows only", nil)
}

func DefaultPaths() (config, work, binary string) {
	return "", "", ""
}

func EnsureSingleProcess(ctx context.Context, exec gitexec.Exec) error {
	return nil
}
