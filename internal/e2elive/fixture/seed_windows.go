//go:build e2e_live && windows

package fixture

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

func seedBadService(ctx context.Context, exec gitexec.Exec, st *State) error {
	bin := st.RunnerBin
	args := []string{
		"install",
		"--service", st.ServiceName,
		"--config", st.ConfigPath,
		"--working-directory", st.WorkDir,
	}
	_, err := exec.Run(ctx, bin, args...)
	if err != nil {
		return wrap("seedBadService", errdefs.CodeServiceStart, "gitlab-runner install (LocalSystem seed)", err)
	}
	st.UseBrew = false
	return nil
}

func disposePlatform(ctx context.Context, exec gitexec.Exec, st *State) error {
	return nil
}

func ensureWindowsPassword(ctx context.Context, exec gitexec.Exec) (string, error) {
	if pw := strings.TrimSpace(os.Getenv("RUNNERCONCIERGE_WINDOWS_PASSWORD")); pw != "" {
		return pw, nil
	}
	if !allowWindowsPasswordReset() {
		return "", wrap("ensureWindowsPassword", errdefs.CodeServiceLogon,
			"Windows password required for repair; set RUNNERCONCIERGE_WINDOWS_PASSWORD or E2E_LIVE_SET_WINDOWS_PASSWORD=1 (or run on GITHUB_ACTIONS)", nil)
	}
	pw, err := randomPassword()
	if err != nil {
		return "", err
	}
	user := strings.TrimSpace(os.Getenv("USERNAME"))
	if user == "" {
		return "", fmt.Errorf("fixture: USERNAME empty")
	}
	_, err = exec.Run(ctx, "net", "user", user, pw)
	if err != nil {
		return "", wrap("ensureWindowsPassword", errdefs.CodeServiceLogon, "net user password reset for fixture user", err)
	}
	return pw, nil
}

func randomPassword() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func expectedBlockingCodes() []errdefs.Code {
	return []errdefs.Code{errdefs.CodeBadServiceLogon}
}
