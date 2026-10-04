//go:build e2e_live && darwin

package fixture

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runSudo(ctx context.Context, bin string, args ...string) error {
	noPass := exec.CommandContext(ctx, "sudo", "-n", bin)
	noPass.Args = append(noPass.Args, args...)
	if err := noPass.Run(); err == nil {
		return nil
	}
	pw := strings.TrimSpace(os.Getenv("E2E_LIVE_SUDO_PASSWORD"))
	if pw == "" {
		return ErrSkipped
	}
	cmd := exec.CommandContext(ctx, "sudo", "-S", bin)
	cmd.Args = append(cmd.Args, args...)
	cmd.Stdin = strings.NewReader(pw + "\n")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo %s: %w (%s)", bin, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
