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

// ensure fakeExec satisfies interface at compile time
var _ gitexec.Exec = (*fakeExec)(nil)
