package cleanup

import (
	"context"
	"errors"
	"io"
	"runtime"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

// EnsureSchedule installs the periodic docker cleanup helper on macOS and Windows.
// On other OSes it is a no-op. allowYes replaces legacy script-based LaunchAgents.
func EnsureSchedule(ctx context.Context, exec gitexec.Exec, out io.Writer, allowYes bool) error {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return nil
	}
	if out == nil {
		out = io.Discard
	}
	ctrl, err := Config{Exec: exec, Out: out, AllowYes: allowYes}.Create()
	if err != nil {
		return err
	}
	if err := ctrl.Install(ctx); err != nil {
		var de *errdefs.Error
		if errors.As(err, &de) && de.Code == errdefs.CodeUnsupportedOS {
			return nil
		}
		return err
	}
	return nil
}
