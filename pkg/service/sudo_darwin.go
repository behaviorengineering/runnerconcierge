//go:build darwin

package service

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
)

func darwinRunSudo(ctx context.Context, cli gitexec.Exec, bin string, args ...string) error {
	sudoArgs := append([]string{"-n", bin}, args...)
	if _, err := cli.Run(ctx, "sudo", sudoArgs...); err == nil {
		return nil
	}
	pw := strings.TrimSpace(os.Getenv("E2E_LIVE_SUDO_PASSWORD"))
	if pw == "" {
		pw = strings.TrimSpace(os.Getenv("RUNNERCONCIERGE_SUDO_PASSWORD"))
	}
	if pw == "" {
		_, err := cli.Run(ctx, "sudo", sudoArgs...)
		return err
	}
	cmd := exec.CommandContext(ctx, "sudo", "-S", bin)
	cmd.Args = append(cmd.Args, args...)
	cmd.Stdin = strings.NewReader(pw + "\n")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	return cmd.Run()
}
