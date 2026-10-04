package detect

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/cliformat"
)

// Render writes a human-readable doctor report.
func Render(w io.Writer, rep *Report) error {
	if rep == nil {
		return fmt.Errorf("detect: report is nil")
	}
	if err := cliformat.Writeln(w, "Host"); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, "  OS:                    %s/%s", rep.GOOS, rep.GOARCH); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, "  Login user:            %s", cliformat.EmptyDash(rep.Username)); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, "  Elevated:              %s", cliformat.YesNo(rep.Elevated)); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, "  gitlab-runner procs:   %d", rep.RunnerProcs); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, ""); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, "Tools"); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, "  Preflight for setup. Use status for inventory and service smells."); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, "  git:                   %s", cliformat.YesNo(rep.GitOK)); err != nil {
		return err
	}
	if strings.TrimSpace(rep.RunnerPath) != "" {
		if err := cliformat.Writeln(w, "  gitlab-runner path:    %s", rep.RunnerPath); err != nil {
			return err
		}
	}
	if err := cliformat.Writeln(w, "  gitlab-runner:         %s", cliformat.CompactVersion(rep.RunnerVer)); err != nil {
		return err
	}
	if strings.TrimSpace(rep.GlabPath) != "" {
		if err := cliformat.Writeln(w, "  glab path:             %s", rep.GlabPath); err != nil {
			return err
		}
	}
	if err := cliformat.Writeln(w, "  glab:                  %s", cliformat.CompactVersion(rep.GlabVer)); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, "  brew:                  %s", cliformat.YesNo(rep.BrewOK)); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, "  winget:                %s", cliformat.YesNo(rep.WingetOK)); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, "  docker reachable:      %s", cliformat.YesNo(rep.DockerOK)); err != nil {
		return err
	}
	if len(rep.ConfigPaths) > 0 {
		if err := cliformat.Writeln(w, ""); err != nil {
			return err
		}
		if err := cliformat.Writeln(w, "Config paths checked"); err != nil {
			return err
		}
		for _, p := range rep.ConfigPaths {
			if err := cliformat.Writeln(w, "  - %s", p); err != nil {
				return err
			}
		}
	}
	if err := cliformat.Writeln(w, ""); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, "Issues"); err != nil {
		return err
	}
	if err := cliformat.Writeln(w, "  block=yes means setup should stop until resolved."); err != nil {
		return err
	}
	if len(rep.Issues) == 0 {
		if err := cliformat.Writeln(w, "  (none)"); err != nil {
			return err
		}
		return nil
	}
	for _, iss := range rep.Issues {
		if err := cliformat.Writeln(w, "  - [%s]  block=%s", iss.Code, cliformat.YesNo(iss.Block)); err != nil {
			return err
		}
		if err := cliformat.Writeln(w, "      %s", iss.Message); err != nil {
			return err
		}
	}
	return nil
}

// RenderJSON writes the doctor report as JSON.
func RenderJSON(w io.Writer, rep *Report) error {
	if rep == nil {
		return fmt.Errorf("detect: report is nil")
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rep)
}

// HasBlocking reports whether any issue blocks setup.
func HasBlocking(rep *Report) bool {
	if rep == nil {
		return false
	}
	for _, iss := range rep.Issues {
		if iss.Block {
			return true
		}
	}
	return false
}
