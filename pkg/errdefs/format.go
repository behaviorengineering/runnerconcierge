package errdefs

import (
	"errors"
	"fmt"
)

// FormatCLI renders one operator-facing line without unwrapping causes into argv dumps.
func FormatCLI(err error) string {
	if err == nil {
		return ""
	}
	var de *Error
	if errors.As(err, &de) && de != nil {
		return de.Error()
	}
	return fmt.Sprintf("error: %s", err.Error())
}

// CodeOf returns the domain code from err when present.
func CodeOf(err error) Code {
	var de *Error
	if errors.As(err, &de) && de != nil && de.Code != "" {
		return de.Code
	}
	return CodeCreateFailed
}
