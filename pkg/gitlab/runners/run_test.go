package runners

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/behaviorengineering/runnerconcierge/pkg/gitlabrunner"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

type recordExec struct {
	calls [][]string
}

func (r *recordExec) LookPath(name string) (string, error) { return "/bin/gitlab-runner", nil }
func (r *recordExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	return nil, nil
}
func (r *recordExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.Run(ctx, name, args...)
}

type stubSvc struct {
	uninstalls int
}

func (s *stubSvc) Install(ctx context.Context, opts service.InstallOpts) error    { return nil }
func (s *stubSvc) Start(ctx context.Context, opts service.StartOpts) error        { return nil }
func (s *stubSvc) Stop(ctx context.Context, opts service.StopOpts) error          { return nil }
func (s *stubSvc) Status(ctx context.Context) (string, error)                     { return "", nil }
func (s *stubSvc) ListOwnership(ctx context.Context) ([]service.Ownership, error) { return nil, nil }
func (s *stubSvc) Uninstall(ctx context.Context, opts service.UninstallOpts) error {
	s.uninstalls++
	return nil
}

func writeTempConfig(t *testing.T, runners int, gitlabURL string, gitlabID int) string {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if gitlabURL == "" {
		gitlabURL = "https://gitlab.com/"
	}
	var b strings.Builder
	b.WriteString("concurrent = 1\n")
	for i := 0; i < runners; i++ {
		name := "runner"
		if runners > 1 {
			name = "runner" + string(rune('a'+i))
		}
		b.WriteString("\n[[runners]]\n")
		if gitlabID > 0 && runners == 1 {
			fmt.Fprintf(&b, "  id = %d\n", gitlabID)
		}
		fmt.Fprintf(&b, "  name = \"%s\"\n  url = \"%s\"\n  executor = \"shell\"\n", name, gitlabURL)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRemove_localSkipsDelete(t *testing.T) {
	path := writeTempConfig(t, 1, "", 0)
	exec := &recordExec{}
	svc := &stubSvc{}
	var out bytes.Buffer
	cfg := Config{
		Exec:       exec,
		Out:        &out,
		Service:    svc,
		ConfigPath: path,
		Name:       "runner",
		Action:     ActionRemove,
		AllowYes:   true,
		LocalOnly:  true,
	}
	ctrl, err := cfg.Create()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := ctrl.Run(ctx); err != nil {
		t.Fatal(err)
	}
	for _, c := range exec.calls {
		joined := strings.Join(c, " ")
		if strings.Contains(joined, "/api/v4/runners") || strings.Contains(joined, "DELETE") {
			t.Fatalf("unexpected GitLab API call: %v", c)
		}
	}
}

func TestRemove_callsDeleteRunner(t *testing.T) {
	var deleted int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v4/runners":
			_ = json.NewEncoder(w).Encode([]gitlabrunner.RunnerInfo{{ID: 9, Name: "runner"}})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v4/runners/9":
			deleted++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	path := writeTempConfig(t, 1, srv.URL+"/", 9)
	exec := &recordExec{}
	svc := &stubSvc{}
	client := gitlabrunner.NewClient(srv.URL, exec, srv.Client())
	var out bytes.Buffer
	cfg := Config{
		Exec:       exec,
		Out:        &out,
		Service:    svc,
		Client:     client,
		GitLabURL:  srv.URL,
		PAT:        "glpat-test",
		ConfigPath: path,
		Name:       "runner",
		Action:     ActionRemove,
		AllowYes:   true,
	}
	ctrl, err := cfg.Create()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := ctrl.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("delete calls %d", deleted)
	}
	assertUnregisterArgv(t, exec)
}

func assertUnregisterArgv(t *testing.T, exec *recordExec) {
	t.Helper()
	var saw bool
	for _, c := range exec.calls {
		if len(c) < 2 || c[0] != "/bin/gitlab-runner" || c[1] != "unregister" {
			continue
		}
		saw = true
		joined := strings.Join(c, " ")
		if strings.Contains(joined, "--non-interactive") {
			t.Fatalf("unregister must not use --non-interactive: %v", c)
		}
		if !strings.Contains(joined, "--name") {
			t.Fatalf("unregister missing --name: %v", c)
		}
	}
	if !saw {
		t.Fatal("expected gitlab-runner unregister call")
	}
}

func TestRemove_unregisterArgv(t *testing.T) {
	path := writeTempConfig(t, 1, "https://gitlab.example/", 0)
	exec := &recordExec{}
	svc := &stubSvc{}
	var out bytes.Buffer
	cfg := Config{
		Exec:       exec,
		Out:        &out,
		Service:    svc,
		ConfigPath: path,
		Name:       "runner",
		Action:     ActionRemove,
		AllowYes:   true,
		LocalOnly:  true,
	}
	ctrl, err := cfg.Create()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := ctrl.Run(ctx); err != nil {
		t.Fatal(err)
	}
	assertUnregisterArgv(t, exec)
}
