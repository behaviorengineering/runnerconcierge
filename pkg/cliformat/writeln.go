package cliformat

import (
	"fmt"
	"io"
)

// EmptyDash returns val or "-" when val is empty.
func EmptyDash(val string) string {
	if val == "" {
		return "-"
	}
	return val
}

// Writeln writes a formatted line with a trailing newline.
func Writeln(w io.Writer, format string, args ...any) error {
	if w == nil {
		return fmt.Errorf("cliformat: writer is nil")
	}
	_, err := fmt.Fprintf(w, format+"\n", args...)
	return err
}
