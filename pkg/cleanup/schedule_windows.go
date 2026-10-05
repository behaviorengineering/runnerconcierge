//go:build windows

package cleanup

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

func platformInstallSchedule(ctx context.Context, exec gitexec.Exec, self string, interval time.Duration, allowYes bool, out io.Writer) error {
	minutes := int(interval.Minutes())
	if minutes < 1 {
		minutes = 1
	}
	tr := quoteWindows(self + " cleanup")
	_, err := exec.Run(ctx, "schtasks", "/Create", "/F", "/SC", "MINUTE", "/MO", fmt.Sprintf("%d", minutes),
		"/TN", UnitName, "/TR", tr, "/RL", "LIMITED")
	if err != nil {
		return errdefs.New("cleanup.Install", errdefs.CodeServiceStart, "schtasks create", err)
	}
	if out != nil {
		fmt.Fprintf(out, "Installed scheduled task %s (every %d minute(s))\n", UnitName, minutes)
	}
	return nil
}

func platformUninstallSchedule(ctx context.Context, exec gitexec.Exec, out io.Writer) error {
	_, err := exec.Run(ctx, "schtasks", "/Delete", "/F", "/TN", UnitName)
	if err != nil && !schtasksMissing(err) {
		return errdefs.New("cleanup.Uninstall", errdefs.CodeServiceStart, "schtasks delete", err)
	}
	if out != nil {
		fmt.Fprintf(out, "Uninstalled scheduled task %s\n", UnitName)
	}
	return nil
}

func quoteWindows(s string) string {
	if strings.ContainsAny(s, " \t\"") {
		return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
	}
	return s
}

func schtasksMissing(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "cannot find the file") ||
		strings.Contains(strings.ToLower(err.Error()), "does not exist")
}
