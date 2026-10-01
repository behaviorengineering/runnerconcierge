package redact

import "regexp"

var (
	glrtToken = regexp.MustCompile(`glrt-[A-Za-z0-9_-]+`)
	glpat     = regexp.MustCompile(`glpat-[A-Za-z0-9_-]+`)
)

const placeholder = "***"

// String redacts known secret patterns in text.
func String(s string) string {
	if s == "" {
		return s
	}
	out := glrtToken.ReplaceAllString(s, "glrt-"+placeholder)
	out = glpat.ReplaceAllString(out, "glpat-"+placeholder)
	return out
}
