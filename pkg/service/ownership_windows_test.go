//go:build windows

package service

import "testing"

func TestParseWindowsOwnershipJSON_identity(t *testing.T) {
	raw := `[{"Name":"gitlab-runner","State":"Running","StartName":".\\user","PathName":"C:\\gitlab-runner.exe run --config C:\\cfg.toml"}]`
	list, err := parseWindowsOwnershipJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("len %d", len(list))
	}
	if list[0].Command == "" {
		t.Fatal("expected command")
	}
	if list[0].MatchReason != MatchReasonWindowsServiceName {
		t.Fatalf("reason %q", list[0].MatchReason)
	}
	if !list[0].ProcessUp {
		t.Fatal("expected process up")
	}
	if list[0].ConfigPath != "C:\\cfg.toml" {
		t.Fatalf("config %q", list[0].ConfigPath)
	}
}
