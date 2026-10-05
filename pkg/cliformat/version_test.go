package cliformat

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteVersion(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteVersion(&buf, "tool", "1.2.3"); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "tool 1.2.3" {
		t.Fatalf("got %q", buf.String())
	}
}

func TestEmptyDash(t *testing.T) {
	if EmptyDash("") != "-" {
		t.Fatal("empty")
	}
	if EmptyDash("x") != "x" {
		t.Fatal("non-empty")
	}
}
