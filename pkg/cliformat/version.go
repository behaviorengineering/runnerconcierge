package cliformat

import "io"

// WriteVersion writes "name version\n" to w.
func WriteVersion(w io.Writer, name, version string) error {
	return Writeln(w, "%s %s", name, version)
}
