package inventory

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

func classifyOwnership(loginUser string, svc service.Ownership, cfgPath string) []Finding {
	var out []Finding
	switch runtime.GOOS {
	case "windows":
		if service.IsLocalSystemLogon(svc.LogonUser) {
			out = append(out, Finding{
				Code:       errdefs.CodeBadServiceLogon,
				Block:      true,
				Message:    fmt.Sprintf("service %q runs as LocalSystem; use login user for keyring access", svc.ServiceName),
				ConfigPath: cfgPath,
				Service:    svc.ServiceName,
				Repairable: true,
			})
		} else if !service.UsersMatch(loginUser, svc.LogonUser) {
			out = append(out, Finding{
				Code:       errdefs.CodeBadServiceLogon,
				Block:      true,
				Message:    fmt.Sprintf("service %q logon %q does not match login user %q", svc.ServiceName, svc.LogonUser, loginUser),
				ConfigPath: cfgPath,
				Service:    svc.ServiceName,
				Repairable: true,
			})
		}
	case "darwin":
		if svc.LogonUser != "" && !service.UsersMatch(loginUser, svc.LogonUser) {
			out = append(out, Finding{
				Code:       errdefs.CodeWrongServiceUser,
				Block:      true,
				Message:    fmt.Sprintf("service %q runs as %q; expected login user %q", svc.ServiceName, svc.LogonUser, loginUser),
				ConfigPath: cfgPath,
				Service:    svc.ServiceName,
				Repairable: true,
			})
		}
	}
	state := strings.ToLower(svc.State)
	if state == "stopped" || strings.Contains(state, "stopped") {
		out = append(out, Finding{
			Code:       errdefs.CodeServiceStopped,
			Block:      false,
			Message:    fmt.Sprintf("service %q is stopped", svc.ServiceName),
			ConfigPath: cfgPath,
			Service:    svc.ServiceName,
			Repairable: false,
		})
	}
	return out
}

func classifyConfigPath(path string, readable bool, elevated bool) []Finding {
	if !service.IsSystemConfigPath(path) {
		return nil
	}
	if !readable && !elevated {
		return []Finding{{
			Code:       errdefs.CodeConfigUnreadable,
			Block:      false,
			Message:    fmt.Sprintf("system config %q exists but is not readable; re-run elevated", path),
			ConfigPath: path,
			Repairable: true,
		}, {
			Code:       errdefs.CodeElevationRequired,
			Block:      false,
			Message:    "elevation required to read system runner config",
			ConfigPath: path,
			Repairable: true,
		}}
	}
	return []Finding{{
		Code:       errdefs.CodeSystemConfig,
		Block:      true,
		Message:    fmt.Sprintf("config %q is system-scoped; runner should use login-user path for keyring", path),
		ConfigPath: path,
		Repairable: true,
	}}
}
