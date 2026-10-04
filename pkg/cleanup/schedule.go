package cleanup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
)

// Install registers the periodic cleanup helper on this OS.
func (c *Controller) Install(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("cleanup: controller is nil")
	}
	self, err := c.selfPath()
	if err != nil {
		return err
	}
	return installSchedule(ctx, c.cfg.Exec, self, c.cfg.Interval, c.cfg.AllowYes, c.cfg.Out)
}

// Uninstall removes the periodic cleanup helper.
func (c *Controller) Uninstall(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("cleanup: controller is nil")
	}
	return uninstallSchedule(ctx, c.cfg.Exec, c.cfg.Out)
}

func (c *Controller) selfPath() (string, error) {
	if strings.TrimSpace(c.cfg.SelfPath) != "" {
		return filepath.Abs(c.cfg.SelfPath)
	}
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(self)
}

func installSchedule(ctx context.Context, exec gitexec.Exec, self string, interval time.Duration, allowYes bool, out io.Writer) error {
	return platformInstallSchedule(ctx, exec, self, interval, allowYes, out)
}

func uninstallSchedule(ctx context.Context, exec gitexec.Exec, out io.Writer) error {
	return platformUninstallSchedule(ctx, exec, out)
}
