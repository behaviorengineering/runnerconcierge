package inventory

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/redact"
)

// Render writes a human-readable status report.
func Render(w io.Writer, rep *Report) error {
	if rep == nil {
		return fmt.Errorf("inventory: report is nil")
	}
	fmt.Fprintf(w, "os=%s arch=%s login_user=%s elevated=%v runner=%s\n\n",
		rep.GOOS, rep.GOARCH, rep.LoginUser, rep.Elevated, rep.RunnerVer)
	for _, c := range rep.Configs {
		fmt.Fprintf(w, "config: %s readable=%v system=%v runners=%d\n", c.Path, c.Readable, c.SystemPath, len(c.Runners))
		for _, r := range c.Runners {
			fmt.Fprintf(w, "  - name=%s url=%s executor=%s\n", r.Name, redact.String(r.URL), r.Executor)
		}
	}
	for _, s := range rep.Services {
		fmt.Fprintf(w, "service: name=%s kind=%s state=%s logon=%s config=%s\n",
			s.ServiceName, s.Kind, s.State, s.LogonUser, s.ConfigPath)
	}
	if len(rep.Findings) > 0 {
		fmt.Fprintln(w, "\nfindings:")
		for _, f := range rep.Findings {
			fmt.Fprintf(w, "  [%s] block=%v repairable=%v %s\n", f.Code, f.Block, f.Repairable, redact.String(f.Message))
		}
	}
	if len(rep.NextActions) > 0 {
		fmt.Fprintln(w, "\nnext actions:")
		for _, a := range rep.NextActions {
			fmt.Fprintf(w, "  - %s\n", a)
		}
	}
	return nil
}

// RenderJSON writes the report as JSON (no runner tokens).
func RenderJSON(w io.Writer, rep *Report) error {
	if rep == nil {
		return fmt.Errorf("inventory: report is nil")
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rep)
}

// HasBlocking reports whether any finding blocks.
func HasBlocking(rep *Report) bool {
	if rep == nil {
		return false
	}
	for _, f := range rep.Findings {
		if f.Block {
			return true
		}
	}
	return false
}

// FindingsForService returns findings tied to a config path.
func FindingsForConfig(rep *Report, configPath string) []Finding {
	if rep == nil {
		return nil
	}
	var out []Finding
	for _, f := range rep.Findings {
		if f.ConfigPath == configPath || strings.TrimSpace(f.ConfigPath) == "" {
			out = append(out, f)
		}
	}
	return out
}
