package state

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCheckpointGroupPathJSON(t *testing.T) {
	cp := &Checkpoint{
		Version:   1,
		GroupPath: "my-group",
		Tags:      []string{"lab"},
	}
	data, err := json.Marshal(cp)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"group_path":"my-group"`) {
		t.Fatalf("json %s", data)
	}
	var back Checkpoint
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back.GroupPath != "my-group" {
		t.Fatalf("group %q", back.GroupPath)
	}
}

func TestCheckpointMarshalNoGLRT(t *testing.T) {
	cp := &Checkpoint{
		Version:     1,
		Description: "host runner",
		Tags:        []string{"ci"},
	}
	data, err := json.Marshal(cp)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "glrt-") {
		t.Fatalf("checkpoint must not contain glrt: %s", data)
	}
}

func TestCheckpointUnmarshalWithoutGroupPath(t *testing.T) {
	const raw = `{"version":1,"tags":["a"]}`
	var cp Checkpoint
	if err := json.Unmarshal([]byte(raw), &cp); err != nil {
		t.Fatal(err)
	}
	if cp.GroupPath != "" {
		t.Fatalf("group %q", cp.GroupPath)
	}
}
