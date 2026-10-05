//go:build e2e_live && (darwin || windows)

package fixture

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func assertCLIStatus(ctx context.Context, cliPath, configPath string, wantExit int) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, cliPath, "status", "--json", "--runner-config", configPath)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	exit := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else {
			return fmt.Errorf("fixture: status cli: %w", err)
		}
	}
	if exit != wantExit {
		return fmt.Errorf("fixture: status cli exit %d want %d: %s", exit, wantExit, truncateOut(out))
	}
	return nil
}

func truncateOut(b []byte) string {
	const max = 512
	s := string(b)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
