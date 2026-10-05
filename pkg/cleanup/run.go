package cleanup

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

// Run removes stale GitLab docker-executor containers and dangling runner-* volumes.
func (c *Controller) Run(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("cleanup: controller is nil")
	}
	now := c.cfg.Clock()
	writeLine(c.cfg.Out, "%s cleanup started", now.Format("2006-01-02 15:04:05"))

	bin, err := resolveDockerBin(ctx, c.cfg.Exec)
	if err != nil {
		writeLine(c.cfg.Out, "Docker is unavailable; skipped")
		return nil
	}
	if _, err := c.cfg.Exec.Run(ctx, bin, "info"); err != nil {
		writeLine(c.cfg.Out, "Docker is unavailable; skipped")
		return nil
	}

	removed := 0
	containers, err := c.cfg.Exec.Run(ctx, bin, "ps", "-aq", "--filter", "status=exited", "--filter", "name=runner-")
	if err != nil {
		return errdefs.New("cleanup.Run", errdefs.CodeDockerUnavailable, "docker ps", err)
	}
	for _, id := range splitLines(string(containers)) {
		if id == "" {
			continue
		}
		finishedAt, ok := c.containerFinishedAt(ctx, bin, id)
		if !ok {
			continue
		}
		if now.Sub(finishedAt) < c.cfg.MinAge {
			continue
		}
		if _, err := c.cfg.Exec.Run(ctx, bin, "rm", "-v", id); err != nil {
			writeLine(c.cfg.Out, "Skipped stopped container %s", id)
			continue
		}
		writeLine(c.cfg.Out, "Removed stopped Runner container and anonymous volumes %s", id)
		removed++
	}

	vols, err := c.cfg.Exec.Run(ctx, bin, "volume", "ls", "-q", "--filter", "dangling=true")
	if err != nil {
		return errdefs.New("cleanup.Run", errdefs.CodeDockerUnavailable, "docker volume ls", err)
	}
	for _, vol := range splitLines(string(vols)) {
		if vol == "" || !strings.HasPrefix(vol, "runner-") {
			continue
		}
		if _, err := c.cfg.Exec.Run(ctx, bin, "volume", "rm", vol); err != nil {
			writeLine(c.cfg.Out, "Skipped %s", vol)
			continue
		}
		writeLine(c.cfg.Out, "Removed %s", vol)
		removed++
	}

	writeLine(c.cfg.Out, "Removed %d Runner resource(s)", removed)
	return nil
}

func (c *Controller) containerFinishedAt(ctx context.Context, bin, id string) (time.Time, bool) {
	out, err := c.cfg.Exec.Run(ctx, bin, "inspect", "--format", "{{.State.FinishedAt}}", id)
	if err != nil {
		return time.Time{}, false
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" || raw == "0001-01-01T00:00:00Z" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339, raw)
	}
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func writeLine(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, format+"\n", args...)
}
