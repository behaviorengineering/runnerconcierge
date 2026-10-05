package config

import (
	"os"
	"testing"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
)

func TestStoreLoadRunnerToken(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	const tok = "glrt-roundtrip"
	if err := StoreRunnerToken(42, tok, mem); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRunnerToken(42, mem)
	if err != nil {
		t.Fatal(err)
	}
	if got != tok {
		t.Fatalf("got %q want %q", got, tok)
	}
}

func TestLoadRunnerTokenMissing(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	got, err := LoadRunnerToken(99, mem)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestLoadRunnerTokenEnvWins(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	env, err := RunnerTokenEnv(7)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(env, "glrt-from-env")
	if err := StoreRunnerToken(7, "glrt-keyring", mem); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRunnerToken(7, mem)
	if err != nil {
		t.Fatal(err)
	}
	if got != "glrt-from-env" {
		t.Fatalf("got %q want env value", got)
	}
}

func TestStoreRunnerTokenInvalidID(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	if err := StoreRunnerToken(0, "glrt-x", mem); err == nil {
		t.Fatal("expected error for id 0")
	}
}

func TestStoreRunnerTokenEmpty(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	if err := StoreRunnerToken(1, "  \n", mem); err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestStoreLoadRunnerTokenByIdentity(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	const tok = "glrt-ident"
	const identity = "behaviorengineering-macos-macstudio"
	if err := StoreRunnerTokenByIdentity(identity, tok, mem); err != nil {
		t.Fatal(err)
	}
	env, err := RunnerTokenEnvForIdentity(identity)
	if err != nil {
		t.Fatal(err)
	}
	if env != "GITLAB_RUNNER_TOKEN_behaviorengineering-macos-macstudio" {
		t.Fatalf("env %q", env)
	}
	got, err := LoadRunnerTokenByIdentity(identity, mem)
	if err != nil || got != tok {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestStoreLoadRunnerTokenByTag(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	const tok = "glrt-tag"
	if err := StoreRunnerTokenByTag("homelab-mac", tok, mem); err != nil {
		t.Fatal(err)
	}
	env, err := RunnerTokenEnvForTag("homelab-mac")
	if err != nil {
		t.Fatal(err)
	}
	if env != "GITLAB_RUNNER_TOKEN_homelab-mac" {
		t.Fatalf("env %q", env)
	}
	got, err := LoadRunnerTokenByTag("homelab-mac", mem)
	if err != nil {
		t.Fatal(err)
	}
	if got != tok {
		t.Fatalf("got %q", got)
	}
}

func TestStoreLoadPendingRunnerToken(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	const tok = "glrt-pending"
	if err := StorePendingRunnerToken(tok, mem); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPendingRunnerToken(mem)
	if err != nil {
		t.Fatal(err)
	}
	if got != tok {
		t.Fatalf("got %q want %q", got, tok)
	}
}

func TestStorePATSetsEnv(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	_ = os.Unsetenv(patAccount)
	if err := StorePAT("glpat-test", mem); err != nil {
		t.Fatal(err)
	}
	if os.Getenv(patAccount) != "glpat-test" {
		t.Fatalf("env %q", os.Getenv(patAccount))
	}
	got, err := mem.Get(appName, patAccount)
	if err != nil {
		t.Fatal(err)
	}
	if got != "glpat-test" {
		t.Fatalf("keyring got %q", got)
	}
}
