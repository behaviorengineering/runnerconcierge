package gitlabrunner

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

type membershipFakeExec struct{}

func (m *membershipFakeExec) LookPath(name string) (string, error) { return name, nil }
func (m *membershipFakeExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return m.Run(ctx, name, args...)
}
func (m *membershipFakeExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name != "glab" || len(args) < 2 || args[0] != "api" {
		return nil, nil
	}
	switch args[1] {
	case "groups/acme":
		return json.Marshal(map[string]int{"id": 10})
	case "groups/10/members/all/7":
		body, _ := json.Marshal(map[string]int{"access_level": 30})
		return body, nil
	case "groups/11/members/all/7":
		body, _ := json.Marshal(map[string]int{"access_level": 40})
		return body, nil
	}
	return nil, nil
}

func TestEnsureCanCreateRunner_insufficientAccess(t *testing.T) {
	client := NewClient("https://gitlab.com", &membershipFakeExec{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := client.EnsureCanCreateRunner(ctx, "group_type", "acme", GitLabUser{ID: 7, Username: "u"})
	if err == nil {
		t.Fatal("expected error")
	}
	if errdefs.CodeOf(err) != errdefs.CodeAuthScopeInsufficient {
		t.Fatalf("code %v", errdefs.CodeOf(err))
	}
}

func TestMemberAccessLevel_maintainer(t *testing.T) {
	client := NewClient("https://gitlab.com", &membershipFakeExec{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	level, err := client.MemberAccessLevel(ctx, "groups", 11, 7)
	if err != nil || level != 40 {
		t.Fatalf("level=%d err=%v", level, err)
	}
}
