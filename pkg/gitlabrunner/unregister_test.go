package gitlabrunner

import "testing"

func TestBuildUnregisterArgv(t *testing.T) {
	args, err := BuildUnregisterArgv(UnregisterArgs{
		Name:       "my-host",
		ConfigPath: "/cfg.toml",
		URL:        "https://gitlab.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if args[0] != "unregister" || args[2] != "--name" || args[3] != "my-host" {
		t.Fatalf("unexpected argv: %v", args)
	}
}

func TestBuildUnregisterArgv_rejectsEmptyName(t *testing.T) {
	_, err := BuildUnregisterArgv(UnregisterArgs{Name: ""})
	if err == nil {
		t.Fatal("expected error")
	}
}
