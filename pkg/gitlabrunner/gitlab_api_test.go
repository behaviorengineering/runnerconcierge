package gitlabrunner

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

func TestValidateCreateRunnerRequestProjectType(t *testing.T) {
	err := ValidateCreateRunnerRequest(CreateRunnerRequest{RunnerType: "project_type"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "project_id") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestCreateRunnerHTTP400Message(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"project_id is missing"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, &fakeExec{}, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, err := client.CreateRunner(ctx, CreateRunnerRequest{
		RunnerType:  "project_type",
		ProjectID:   99,
		Description: "test",
	}, "glpat-test")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "glab api --method") {
		t.Fatalf("leaked argv: %v", err)
	}
	if !strings.Contains(err.Error(), "project_id is missing") {
		t.Fatalf("expected GitLab message: %v", err)
	}
}

type glabFailExec struct{}

func (g *glabFailExec) LookPath(name string) (string, error) { return name, nil }
func (g *glabFailExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return nil, errors.New(`glab: HTTP 400: {"message":"bad request"}`)
}
func (g *glabFailExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return nil, nil
}

func TestCreateRunnerGlabDoesNotLeakArgv(t *testing.T) {
	client := NewClient("https://gitlab.example.invalid", &glabFailExec{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, err := client.CreateRunner(ctx, CreateRunnerRequest{
		RunnerType:  "project_type",
		ProjectID:   1,
		Description: "x",
	}, "")
	if err == nil {
		t.Fatal("expected error")
	}
	var de *errdefs.Error
	if !errors.As(err, &de) {
		t.Fatalf("expected domain error: %v", err)
	}
	if strings.Contains(de.Error(), "glab api --method") {
		t.Fatalf("leaked argv: %q", de.Error())
	}
	if !strings.Contains(de.Error(), "bad request") {
		t.Fatalf("expected parsed message: %q", de.Error())
	}
}

var _ gitexec.Exec = (*glabFailExec)(nil)
