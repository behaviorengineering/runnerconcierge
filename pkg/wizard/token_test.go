package wizard

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

func testRunnerIdentity(parent, tag string) (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return RunnerIdentity(parent, tag, host)
}

func TestResolveRegisterTokenFromKeyring(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	const tok = "glrt-resume"
	identity, err := testRunnerIdentity("acme", "lab")
	if err != nil {
		t.Fatal(err)
	}
	if err := config.StoreRunnerTokenByIdentity(identity, tok, mem); err != nil {
		t.Fatal(err)
	}
	cp := &state.Checkpoint{RunnerID: 55, GroupPath: "acme", Tags: []string{"lab"}}
	opts := Options{GroupPath: "acme", TagList: []string{"lab"}}
	got, id, err := resolveRegisterToken(context.Background(), opts, cp, mem, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != tok || id != 55 {
		t.Fatalf("got token=%q id=%d", got, id)
	}
}

func TestResolveRegisterTokenMissingWithID(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	cp := &state.Checkpoint{RunnerID: 12, GroupPath: "acme", Tags: []string{"lab"}}
	opts := Options{GroupPath: "acme", TagList: []string{"lab"}}
	_, _, err := resolveRegisterToken(context.Background(), opts, cp, mem, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if errdefs.CodeOf(err) != errdefs.CodeInvalidScope {
		t.Fatalf("code %v", errdefs.CodeOf(err))
	}
}

func TestResolveRegisterTokenCreatesAndStores(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	cp := &state.Checkpoint{GroupPath: "acme", Tags: []string{"lab"}}
	opts := Options{GroupPath: "acme", TagList: []string{"lab"}}
	identity, err := testRunnerIdentity("acme", "lab")
	if err != nil {
		t.Fatal(err)
	}
	create := func(ctx context.Context) (int, string, error) {
		return 88, "glrt-new", nil
	}
	got, id, err := resolveRegisterToken(context.Background(), opts, cp, mem, create)
	if err != nil {
		t.Fatal(err)
	}
	if got != "glrt-new" || id != 88 || cp.RunnerID != 88 {
		t.Fatalf("token=%q id=%d cp=%d", got, id, cp.RunnerID)
	}
	loaded, err := config.LoadRunnerTokenByIdentity(identity, mem)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != "glrt-new" {
		t.Fatalf("keyring %q", loaded)
	}
}

func TestResolveRegisterTokenFlagStoresIdentity(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	cp := &state.Checkpoint{GroupPath: "acme", Tags: []string{"lab"}}
	opts := Options{RunnerToken: "glrt-flag", GroupPath: "acme", TagList: []string{"lab"}}
	identity, err := testRunnerIdentity("acme", "lab")
	if err != nil {
		t.Fatal(err)
	}
	got, id, err := resolveRegisterToken(context.Background(), opts, cp, mem, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "glrt-flag" || id != 0 {
		t.Fatalf("token=%q id=%d", got, id)
	}
	byID, err := config.LoadRunnerTokenByIdentity(identity, mem)
	if err != nil {
		t.Fatal(err)
	}
	if byID != "glrt-flag" {
		t.Fatalf("identity store %q", byID)
	}
}

func TestResolveRegisterTokenIgnoresTagKeyringWhenNoRunnerID(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	if err := config.StoreRunnerTokenByTag("lab", "glrt-stale", mem); err != nil {
		t.Fatal(err)
	}
	cp := &state.Checkpoint{GroupPath: "acme", Tags: []string{"lab"}}
	opts := Options{GroupPath: "acme", TagList: []string{"lab"}}
	create := func(ctx context.Context) (int, string, error) {
		return 99, "glrt-new", nil
	}
	got, id, err := resolveRegisterToken(context.Background(), opts, cp, mem, create)
	if err != nil {
		t.Fatal(err)
	}
	if got != "glrt-new" || id != 99 {
		t.Fatalf("token=%q id=%d", got, id)
	}
}

func TestLoadStoredRunnerGLRT_skipsTagWhenIdentityKnown(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	if err := config.StoreRunnerTokenByTag("lab", "glrt-stale", mem); err != nil {
		t.Fatal(err)
	}
	cp := &state.Checkpoint{RunnerID: 12, GroupPath: "acme", Tags: []string{"lab"}}
	opts := Options{GroupPath: "acme", TagList: []string{"lab"}}
	got, err := loadStoredRunnerGLRT(opts, cp, mem)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("token %q", got)
	}
}

func TestResolveRegisterTokenCreateError(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	cp := &state.Checkpoint{GroupPath: "acme", Tags: []string{"lab"}}
	opts := Options{GroupPath: "acme", TagList: []string{"lab"}}
	create := func(ctx context.Context) (int, string, error) {
		return 0, "", errors.New("api down")
	}
	_, _, err := resolveRegisterToken(context.Background(), opts, cp, mem, create)
	if err == nil {
		t.Fatal("expected error")
	}
}
