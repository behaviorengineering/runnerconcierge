package runners

import (
	"fmt"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/cliformat"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

func formatPickerLabel(t Target) string {
	if t.Kind == TargetKindRegistration {
		svc := strings.TrimSpace(t.ServiceName)
		if svc == "" {
			svc = "not linked"
		}
		return fmt.Sprintf("registered runner · %s · service %s",
			cliformat.EmptyDash(t.Name), svc)
	}
	role := strings.TrimSpace(t.Role)
	if role == "" {
		role = service.RoleUnknown
	}
	proc := "process down"
	if t.ProcessUp {
		proc = "process up"
	}
	kind := strings.TrimSpace(t.ServiceKind)
	if kind == "" {
		kind = "service"
	}
	return fmt.Sprintf("%s · unit %s · %s · %s",
		role, cliformat.EmptyDash(t.ServiceName), kind, proc)
}
