package runners

import (
	"fmt"
	"io"

	"github.com/behaviorengineering/runnerconcierge/pkg/cliformat"
)

func renderList(w io.Writer, targets []Target) error {
	if err := cliformat.Writeln(w, "Runners"); err != nil {
		return err
	}
	if len(targets) == 0 {
		return cliformat.Writeln(w, "  (none found)")
	}
	for _, t := range targets {
		line := fmt.Sprintf("  - %s  service=%s  state=%s  config=%s",
			cliformat.EmptyDash(t.Name), cliformat.EmptyDash(t.ServiceName),
			cliformat.EmptyDash(t.ServiceState), cliformat.EmptyDash(t.ConfigPath))
		if err := cliformat.Writeln(w, "%s", line); err != nil {
			return err
		}
	}
	return cliformat.Writeln(w, "")
}

func renderTarget(w io.Writer, t *Target, gitlabOnline string) error {
	if t == nil {
		return fmt.Errorf("runners: target is nil")
	}
	if err := cliformat.Writeln(w, "Runner"); err != nil {
		return err
	}
	rows := []struct {
		label string
		val   string
	}{
		{"Name", t.Name},
		{"URL", t.URL},
		{"Executor", t.Executor},
		{"Config path", t.ConfigPath},
		{"Service", t.ServiceName},
		{"Service kind", t.ServiceKind},
		{"Service state", t.ServiceState},
		{"Entries in config", fmt.Sprintf("%d", t.EntryCount)},
	}
	if t.GitLabID > 0 {
		rows = append(rows, struct{ label, val string }{"GitLab id", fmt.Sprintf("%d", t.GitLabID)})
	} else {
		rows = append(rows, struct{ label, val string }{"GitLab id", "(unresolved)"})
	}
	if gitlabOnline != "" {
		rows = append(rows, struct{ label, val string }{"GitLab online", gitlabOnline})
	}
	for _, r := range rows {
		if err := cliformat.Writeln(w, "  %s:  %s", r.label, cliformat.EmptyDash(r.val)); err != nil {
			return err
		}
	}
	return cliformat.Writeln(w, "")
}
