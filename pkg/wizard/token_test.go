package wizard

import (
	"context"
	"errors"
	"testing"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

func TestResolveRegisterTokenFromKeyring(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	const tok = "glrt-resume"
	if err := config.StoreRunnerToken(55, tok, mem); err != nil {
		t.Fatal(err)
	}
	cp := &state.Checkpoint{RunnerID: 55}
	got, id, err := resolveRegisterToken(context.Background(), Options{}, cp, mem, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != tok || id != 55 {
		t.Fatalf("got token=%q id=%d", got, id)
	}
}

func TestResolveRegisterTokenMissingWithID(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	cp := &state.Checkpoint{RunnerID: 12}
	_, _, err := resolveRegisterToken(context.Background(), Options{}, cp, mem, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if errdefs.CodeOf(err) != errdefs.CodeInvalidScope {
		t.Fatalf("code %v", errdefs.CodeOf(err))
	}
}

func TestResolveRegisterTokenCreatesAndStores(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	cp := &state.Checkpoint{}
	create := func(ctx context.Context) (int, string, error) {
		return 88, "glrt-new", nil
	}
	got, id, err := resolveRegisterToken(context.Background(), Options{}, cp, mem, create)
	if err != nil {
		t.Fatal(err)
	}
	if got != "glrt-new" || id != 88 || cp.RunnerID != 88 {
		t.Fatalf("token=%q id=%d cp=%d", got, id, cp.RunnerID)
	}
	loaded, err := config.LoadRunnerToken(88, mem)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != "glrt-new" {
		t.Fatalf("keyring %q", loaded)
	}
}

func TestResolveRegisterTokenFlagStoresPending(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	cp := &state.Checkpoint{}
	got, id, err := resolveRegisterToken(context.Background(), Options{RunnerToken: "glrt-flag", TagList: []string{"lab"}}, cp, mem, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "glrt-flag" || id != 0 {
		t.Fatalf("token=%q id=%d", got, id)
	}
	byTag, err := config.LoadRunnerTokenByTag("lab", mem)
	if err != nil {
		t.Fatal(err)
	}
	if byTag != "glrt-flag" {
		t.Fatalf("tag store %q", byTag)
	}
}

func TestResolveRegisterTokenCreateError(t *testing.T) {
	mem := operatorconfig.NewMemKeyring()
	cp := &state.Checkpoint{}
	create := func(ctx context.Context) (int, string, error) {
		return 0, "", errors.New("api down")
	}
	_, _, err := resolveRegisterToken(context.Background(), Options{}, cp, mem, create)
	if err == nil {
		t.Fatal("expected error")
	}
}
