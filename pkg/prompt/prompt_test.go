package prompt

import (
	"context"
	"testing"
)

func TestNonInteractiveSelect(t *testing.T) {
	var ni NonInteractive
	_, err := ni.Select(context.Background(), "pick", []string{"a", "b"})
	if err == nil {
		t.Fatal("expected error")
	}
}
