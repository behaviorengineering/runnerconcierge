//go:build !darwin && !windows

package cleanup

import (
	"context"
	"io"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

func platformInstallSchedule(ctx context.Context, exec gitexec.Exec, self string, interval time.Duration, allowYes bool, out io.Writer) error {
	return errdefs.New("cleanup.Install", errdefs.CodeUnsupportedOS, "cleanup install supported on darwin and windows only", nil)
}

func platformUninstallSchedule(ctx context.Context, exec gitexec.Exec, out io.Writer) error {
	return errdefs.New("cleanup.Uninstall", errdefs.CodeUnsupportedOS, "cleanup uninstall supported on darwin and windows only", nil)
}
