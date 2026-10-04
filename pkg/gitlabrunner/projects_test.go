package gitlabrunner

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type projectsFakeExec struct {
	argv []string
}

func (p *projectsFakeExec) LookPath(name string) (string, error) { return name, nil }
func (p *projectsFakeExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	p.argv = append([]string{name}, args...)
	if name == "glab" && len(args) >= 2 && args[0] == "api" && stringsHasPrefix(args[1], "projects?") {
		body, _ := json.Marshal([]MemberProject{
			{ID: 1, PathWithNamespace: "acme/widget", Name: "Widget"},
			{ID: 2, PathWithNamespace: "acme/other", Name: "Other"},
		})
		return body, nil
	}
	return nil, nil
}
func (p *projectsFakeExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return p.Run(ctx, name, args...)
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func TestParseGitLabProjectPath(t *testing.T) {
	cases := []struct {
		remote, host, want string
		ok                 bool
	}{
		{"git@gitlab.com:acme/widget.git", "gitlab.com", "acme/widget", true},
		{"https://gitlab.com/acme/widget.git", "gitlab.com", "acme/widget", true},
		{"https://gitlab.com/acme/sub/widget", "gitlab.com", "acme/sub/widget", true},
		{"git@github.com:acme/widget.git", "gitlab.com", "", false},
	}
	for _, tc := range cases {
		got, ok := ParseGitLabProjectPath(tc.remote, tc.host)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("%q: got %q ok=%v want %q %v", tc.remote, got, ok, tc.want, tc.ok)
		}
	}
}

func TestListMemberProjects(t *testing.T) {
	exec := &projectsFakeExec{}
	client := NewClient("https://gitlab.com", exec, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	list, err := client.ListMemberProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].PathWithNamespace != "acme/widget" {
		t.Fatalf("list: %+v", list)
	}
}
