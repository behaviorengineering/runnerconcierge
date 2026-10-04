package runners

import (
	"fmt"
	"io"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/cliformat"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

func renderList(w io.Writer, targets []Target) error {
	if err := cliformat.Writeln(w, "Runners"); err != nil {
		return err
	}
	if len(targets) == 0 {
		return cliformat.Writeln(w, "  (none found)")
	}
	for _, t := range targets {
		if err := cliformat.Writeln(w, "  - %s", formatPickerLabel(t)); err != nil {
			return err
		}
	}
	return cliformat.Writeln(w, "")
}

func inspectTitle(t *Target) string {
	if t == nil {
		return "Target"
	}
	if t.Kind == TargetKindServiceOnly {
		return "Service unit"
	}
	return "Registered runner"
}

func inspectRows(t *Target, gitlabOnline string) []struct {
	label string
	val   string
} {
	if t == nil {
		return nil
	}
	var rows []struct {
		label string
		val   string
	}
	if t.Kind == TargetKindServiceOnly {
		rows = append(rows,
			struct{ label, val string }{"Role", cliformat.EmptyDash(t.Role)},
			struct{ label, val string }{"Why listed", service.HumanMatchReason(t.MatchReason)},
			struct{ label, val string }{"Unit path", cliformat.EmptyDash(t.UnitPath)},
			struct{ label, val string }{"Command", commandForInspect(t.Command)},
			struct{ label, val string }{"Service state", cliformat.EmptyDash(t.ServiceState)},
			struct{ label, val string }{"Process up", processUpText(t.ProcessUp)},
		)
		if strings.TrimSpace(t.ConfigPath) != "" {
			rows = append(rows, struct{ label, val string }{"Config path", t.ConfigPath})
		}
		if strings.TrimSpace(t.LogonUser) != "" {
			rows = append(rows, struct{ label, val string }{"Logon user", t.LogonUser})
		}
		return rows
	}
	rows = append(rows,
		struct{ label, val string }{"Name", t.Name},
		struct{ label, val string }{"URL", t.URL},
		struct{ label, val string }{"Executor", t.Executor},
		struct{ label, val string }{"Config path", t.ConfigPath},
	)
	if strings.TrimSpace(t.ServiceName) != "" {
		rows = append(rows,
			struct{ label, val string }{"Service", t.ServiceName},
			struct{ label, val string }{"Service kind", t.ServiceKind},
			struct{ label, val string }{"Service state", t.ServiceState},
			struct{ label, val string }{"Service role", cliformat.EmptyDash(t.Role)},
			struct{ label, val string }{"Process up", processUpText(t.ProcessUp)},
		)
		if strings.TrimSpace(t.Command) != "" {
			rows = append(rows, struct{ label, val string }{"Service command", commandForInspect(t.Command)})
		}
	}
	rows = append(rows, struct{ label, val string }{"Entries in config", fmt.Sprintf("%d", t.EntryCount)})
	if t.GitLabID > 0 {
		rows = append(rows, struct{ label, val string }{"GitLab id", fmt.Sprintf("%d", t.GitLabID)})
	} else if t.Name != "" {
		rows = append(rows, struct{ label, val string }{"GitLab id", "(unresolved)"})
	}
	if gitlabOnline != "" {
		rows = append(rows, struct{ label, val string }{"GitLab online", gitlabOnline})
	}
	return omitEmptyInspectRows(rows)
}

func omitEmptyInspectRows(rows []struct{ label, val string }) []struct{ label, val string } {
	out := make([]struct{ label, val string }, 0, len(rows))
	for _, r := range rows {
		if strings.TrimSpace(r.val) == "" || r.val == "-" {
			continue
		}
		out = append(out, r)
	}
	return out
}

func commandForInspect(cmd string) string {
	if strings.TrimSpace(cmd) == "" {
		return "(unparsed)"
	}
	return cmd
}

func processUpText(up bool) string {
	if up {
		return "yes"
	}
	return "no"
}

func formatInspectBody(t *Target, gitlabOnline string) string {
	var b strings.Builder
	for _, r := range inspectRows(t, gitlabOnline) {
		b.WriteString(fmt.Sprintf("%s: %s\n", r.label, r.val))
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderTarget(w io.Writer, t *Target, gitlabOnline string) error {
	if t == nil {
		return fmt.Errorf("runners: target is nil")
	}
	if err := cliformat.Writeln(w, "%s", inspectTitle(t)); err != nil {
		return err
	}
	for _, r := range inspectRows(t, gitlabOnline) {
		if err := cliformat.Writeln(w, "  %s:  %s", r.label, r.val); err != nil {
			return err
		}
	}
	return cliformat.Writeln(w, "")
}
