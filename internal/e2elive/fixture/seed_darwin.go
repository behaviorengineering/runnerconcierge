//go:build e2e_live && darwin

package fixture

import (
	"context"
	"errors"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

func seedBadService(ctx context.Context, exec gitexec.Exec, st *State) error {
	// gitlab-runner 19+ requires --user on install; root vs login user yields a repairable smell.
	err := runSudo(ctx, "gitlab-runner", "install",
		"--service", st.ServiceName,
		"--user", "root",
		"--config", st.ConfigPath,
		"--working-directory", st.WorkDir)
	if err != nil {
		if errors.Is(err, ErrSkipped) {
			if envTruthy("E2E_LIVE_DARWIN_USER_SEED") {
				return seedUserService(ctx, exec, st)
			}
			return wrap("seedBadService", errdefs.CodeElevationRequired,
				"darwin fixture needs sudo (passwordless or E2E_LIVE_SUDO_PASSWORD) for a repairable smell alongside an existing runner", err)
		}
		return wrap("seedBadService", errdefs.CodeServiceStart, "sudo gitlab-runner install", err)
	}
	st.UseBrew = false
	st.SystemSeed = true
	return nil
}

func seedUserService(ctx context.Context, exec gitexec.Exec, st *State) error {
	login, _ := service.LoginUser(exec)
	_, err := exec.Run(ctx, "gitlab-runner", "install",
		"--service", st.ServiceName,
		"--user", login,
		"--config", st.ConfigPath,
		"--working-directory", st.WorkDir)
	if err != nil {
		return wrap("seedUserService", errdefs.CodeServiceStart, "gitlab-runner install (user seed)", err)
	}
	st.UseBrew = false
	st.SystemSeed = false
	return nil
}

func ensureWindowsPassword(ctx context.Context, exec gitexec.Exec) (string, error) {
	return "", nil
}

func expectedBlockingCodes() []errdefs.Code {
	return []errdefs.Code{errdefs.CodeWrongServiceUser, errdefs.CodeSystemConfig, errdefs.CodeBadServiceLogon}
}

func disposePlatform(ctx context.Context, exec gitexec.Exec, st *State) error {
	if st == nil || exec == nil || st.ServiceName == "" {
		return nil
	}
	bin := st.RunnerBin
	if bin == "" {
		bin, _ = exec.LookPath("gitlab-runner")
	}
	if bin != "" {
		_ = runSudoUninstall(ctx, bin, st.ServiceName)
	}
	service.ForceRemoveLaunchdPlist(ctx, exec, st.ServiceName)
	service.ForceRemoveLaunchdPlistsWithPrefix(ctx, exec, namePrefix)
	return nil
}
