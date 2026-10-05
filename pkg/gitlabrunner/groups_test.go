package gitlabrunner

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

type groupsFakeExec struct{}

func (g *groupsFakeExec) LookPath(name string) (string, error) { return name, nil }
func (g *groupsFakeExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "glab" && len(args) >= 2 && args[0] == "api" && stringsHasPrefix(args[1], "groups?") {
		if !strings.Contains(args[1], "min_access_level=40") {
			return nil, errors.New("missing min_access_level")
		}
		body, _ := json.Marshal([]MemberGroup{
			{ID: 10, FullPath: "acme", Name: "Acme"},
		})
		return body, nil
	}
	if name == "glab" && len(args) == 2 && args[0] == "api" && args[1] == "groups/acme" {
		body, _ := json.Marshal(map[string]int{"id": 10})
		return body, nil
	}
	return nil, nil
}
func (g *groupsFakeExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return g.Run(ctx, name, args...)
}

func TestListMemberGroups(t *testing.T) {
	client := NewClient("https://gitlab.com", &groupsFakeExec{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	list, err := client.ListMemberGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].FullPath != "acme" {
		t.Fatalf("list: %+v", list)
	}
}

func TestResolveGroupID(t *testing.T) {
	client := NewClient("https://gitlab.com", &groupsFakeExec{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := client.ResolveGroupID(ctx, "acme")
	if err != nil || id != 10 {
		t.Fatalf("id=%d err=%v", id, err)
	}
}
