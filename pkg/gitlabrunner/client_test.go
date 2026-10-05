package gitlabrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
)

type fakeExec struct{}

func (f *fakeExec) LookPath(name string) (string, error) { return name, nil }
func (f *fakeExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return nil, nil
}
func (f *fakeExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return nil, nil
}

func TestCreateRunnerHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/user/runners" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "token": "glrt-test"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, &fakeExec{}, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, tok, err := client.CreateRunner(ctx, CreateRunnerRequest{
		RunnerType:  "project_type",
		ProjectID:   1,
		Description: "test",
		TagList:     []string{"ci"},
	}, "glpat-test")
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 || tok != "glrt-test" {
		t.Fatalf("unexpected %d %s", id, tok)
	}
}

func TestListOwnedRunnersHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/runners" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]RunnerInfo{{ID: 9, Name: "test"}})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, &fakeExec{}, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	list, err := client.ListOwnedRunners(ctx, "glpat-test")
	if err != nil || len(list) != 1 || list[0].ID != 9 {
		t.Fatalf("list: %v err=%v", list, err)
	}
}

func TestDeleteRunnerHTTP(t *testing.T) {
	var deleted bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v4/runners/9" {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, &fakeExec{}, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.DeleteRunner(ctx, 9, "glpat-test"); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected DELETE")
	}
}

func TestListRunnersWrapsGlabError(t *testing.T) {
	client := NewClient("https://gitlab.com", &listRunnersFailExec{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.ListRunners(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "secret argv") {
		t.Fatalf("leaked argv: %v", err)
	}
}

type listRunnersFailExec struct{}

func (g *listRunnersFailExec) LookPath(name string) (string, error) { return name, nil }
func (g *listRunnersFailExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return nil, fmt.Errorf("glab failed with secret argv")
}
func (g *listRunnersFailExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return g.Run(ctx, name, args...)
}

type resetTokenExec struct{}

func (resetTokenExec) LookPath(name string) (string, error) { return name, nil }
func (resetTokenExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return resetTokenExec{}.Run(ctx, name, args...)
}
func (resetTokenExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "glab" && len(args) >= 4 && args[0] == "api" && args[1] == "--method" && args[2] == "POST" &&
		args[3] == "runners/57039107/reset_authentication_token" {
		return []byte(`{"token":"glrt-test"}`), nil
	}
	if name == "glab" && len(args) == 2 && args[0] == "api" && args[1] == "user" {
		return []byte(`{"id":1,"username":"tester"}`), nil
	}
	return nil, nil
}

func TestResetAuthenticationToken_glab(t *testing.T) {
	client := NewClient("https://gitlab.com", resetTokenExec{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tok, err := client.ResetAuthenticationToken(ctx, 57039107)
	if err != nil {
		t.Fatal(err)
	}
	if tok != "glrt-test" {
		t.Fatalf("token %q", tok)
	}
}

func TestResetAuthenticationToken_emptyTokenFails(t *testing.T) {
	client := NewClient("https://gitlab.com", emptyResetExec{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.ResetAuthenticationToken(ctx, 1)
	if err == nil {
		t.Fatal("expected error for empty token response")
	}
}

type emptyResetExec struct{}

func (emptyResetExec) LookPath(name string) (string, error) { return name, nil }
func (emptyResetExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return emptyResetExec{}.Run(ctx, name, args...)
}
func (emptyResetExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return []byte(`{"token":""}`), nil
}

func TestWhoAmI_returnsID(t *testing.T) {
	client := NewClient("https://gitlab.com", resetTokenExec{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	u, err := client.WhoAmI(ctx)
	if err != nil || u.ID != 1 || u.Username != "tester" {
		t.Fatalf("user %+v err %v", u, err)
	}
}

// ensure fakeExec satisfies interface at compile time
var _ gitexec.Exec = (*fakeExec)(nil)
