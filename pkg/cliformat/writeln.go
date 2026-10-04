package cliformat

import (
	"fmt"
	"io"
	"strings"
)

// Writeln prints one logical line with a trailing newline. Empty format prints a blank line.
func Writeln(w io.Writer, format string, args ...any) error {
	if format == "" {
		_, err := fmt.Fprintln(w)
		return err
	}
	out := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	_, err := fmt.Fprint(w, out)
	return err
}

// YesNo formats a bool for operator reports.
func YesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

// EmptyDash returns "-" for blank strings.
func EmptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
