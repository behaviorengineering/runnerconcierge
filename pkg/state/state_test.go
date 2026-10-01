package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreSaveLoad(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runnerconcierge")
	s := &Store{Dir: dir}
	cp := &Checkpoint{
		Version:     1,
		Fingerprint: "abc",
		Stage:       "doctor",
		RunnerID:    7,
	}
	if err := s.Save(cp); err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.Load()
	if err != nil || !ok {
		t.Fatalf("load: ok=%v err=%v", ok, err)
	}
	if got.RunnerID != 7 || got.Fingerprint != "abc" {
		t.Fatalf("unexpected checkpoint: %+v", got)
	}
	data, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if contains(string(data), "glrt-") {
		t.Fatal("checkpoint must not contain tokens")
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && search(s, sub)
}

func search(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
