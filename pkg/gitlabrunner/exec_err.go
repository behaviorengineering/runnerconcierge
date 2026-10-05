package gitlabrunner

import "strings"

// childErrorSummary returns a short operator hint from a failed gitlab-runner/glab exec.
func childErrorSummary(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if strings.Contains(trim, "PANIC:") || strings.Contains(trim, "not valid") || strings.Contains(trim, "FATAL:") {
			return redactGLRT(trim)
		}
	}
	if len(s) > 240 {
		return redactGLRT(s[:240] + "...")
	}
	return redactGLRT(strings.TrimSpace(s))
}

func redactGLRT(s string) string {
	if idx := strings.Index(s, "glrt-"); idx >= 0 {
		return s[:idx] + "glrt-…"
	}
	return s
}
