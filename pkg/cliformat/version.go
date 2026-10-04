package cliformat

import "strings"

// CompactVersion returns one identity token from a tool --version banner.
func CompactVersion(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "(not found)"
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "version:") {
			return strings.TrimSpace(line[len("version:"):])
		}
	}
	return strings.TrimSpace(strings.Split(raw, "\n")[0])
}
