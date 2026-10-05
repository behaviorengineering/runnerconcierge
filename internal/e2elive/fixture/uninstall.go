//go:build e2e_live && (darwin || windows)

package fixture

import (
	"context"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

func uninstallFixtureService(ctx context.Context, exec gitexec.Exec, st *State) error {
	if st == nil || exec == nil || st.ServiceName == "" {
		return nil
	}
	bin := st.RunnerBin
	if bin == "" {
		bin, _ = exec.LookPath("gitlab-runner")
	}
	if bin == "" {
		return nil
	}
	svcMgr := service.New(exec)
	_ = svcMgr.Uninstall(ctx, service.UninstallOpts{
		BinaryPath:  bin,
		ServiceName: st.ServiceName,
		ConfigPath:  st.ConfigPath,
	})
	return nil
}
