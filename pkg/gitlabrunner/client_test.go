package gitlabrunner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// ensure fakeExec satisfies interface at compile time
var _ gitexec.Exec = (*fakeExec)(nil)
