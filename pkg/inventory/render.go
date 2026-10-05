package inventory

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/redact"
)

func writef(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}

// Render writes a human-readable status report.
func Render(w io.Writer, rep *Report) error {
	if rep == nil {
		return fmt.Errorf("inventory: report is nil")
	}
	if err := writef(w, "os=%s arch=%s login_user=%s elevated=%v runner=%s\n\n",
		rep.GOOS, rep.GOARCH, rep.LoginUser, rep.Elevated, rep.RunnerVer); err != nil {
		return err
	}
	for _, c := range rep.Configs {
		if err := writef(w, "config: %s readable=%v system=%v runners=%d\n", c.Path, c.Readable, c.SystemPath, len(c.Runners)); err != nil {
			return err
		}
		for _, r := range c.Runners {
			if err := writef(w, "  - name=%s url=%s executor=%s\n", r.Name, redact.String(r.URL), r.Executor); err != nil {
				return err
			}
		}
	}
	for _, s := range rep.Services {
		if err := writef(w, "service: name=%s kind=%s state=%s logon=%s config=%s\n",
			s.ServiceName, s.Kind, s.State, s.LogonUser, s.ConfigPath); err != nil {
			return err
		}
	}
	if len(rep.Findings) > 0 {
		if _, err := fmt.Fprintln(w, "\nfindings:"); err != nil {
			return err
		}
		for _, f := range rep.Findings {
			if err := writef(w, "  [%s] block=%v repairable=%v %s\n", f.Code, f.Block, f.Repairable, redact.String(f.Message)); err != nil {
				return err
			}
		}
	}
	if len(rep.NextActions) > 0 {
		if _, err := fmt.Fprintln(w, "\nnext actions:"); err != nil {
			return err
		}
		for _, a := range rep.NextActions {
			if err := writef(w, "  - %s\n", a); err != nil {
				return err
			}
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

// HasBlockingForTarget reports blocking findings scoped to one config path and/or service name.
func HasBlockingForTarget(rep *Report, configPath, serviceName string) bool {
	if rep == nil {
		return false
	}
	configPath = strings.TrimSpace(configPath)
	serviceName = strings.TrimSpace(serviceName)
	for _, f := range rep.Findings {
		if !f.Block {
			continue
		}
		if configPath != "" && f.ConfigPath == configPath {
			return true
		}
		if serviceName != "" && f.Service == serviceName {
			return true
		}
	}
	return false
}

// FindingsForConfig returns findings tied to a config path.
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
