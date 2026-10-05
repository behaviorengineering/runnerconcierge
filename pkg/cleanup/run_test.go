package cleanup

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
)

type dockerExec struct {
	dockerBin string
	responses map[string][]byte
	errors    map[string]error
}

func (d *dockerExec) LookPath(name string) (string, error) {
	if name == "docker" {
		return d.dockerBin, nil
	}
	return "", fmt.Errorf("not found")
}

func (d *dockerExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	if d.errors != nil {
		if err, ok := d.errors[key]; ok {
			return nil, err
		}
	}
	if d.responses != nil {
		if out, ok := d.responses[key]; ok {
			return out, nil
		}
	}
	if strings.Contains(key, " inspect --format") {
		if out, ok := d.responses["inspect"]; ok {
			return out, nil
		}
	}
	if strings.HasPrefix(key, d.dockerBin+" rm") {
		return nil, nil
	}
	if strings.HasPrefix(key, d.dockerBin+" volume rm") {
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected: %s", key)
}

func (d *dockerExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return d.Run(ctx, name, args...)
}

func TestRun_skipsWhenDockerDown(t *testing.T) {
	exec := &dockerExec{
		dockerBin: "/bin/docker",
		errors: map[string]error{
			"/bin/docker info": fmt.Errorf("down"),
		},
	}
	var out bytes.Buffer
	ctrl, err := Config{Exec: exec, Out: &out}.Create()
	if err != nil {
		t.Fatal(err)
	}
	if err := ctrl.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "skipped") {
		t.Fatalf("out: %q", out.String())
	}
}

func TestRun_removesOldContainer(t *testing.T) {
	old := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339Nano)
	exec := &dockerExec{
		dockerBin: "/bin/docker",
		responses: map[string][]byte{
			"/bin/docker info": nil,
			"/bin/docker ps -aq --filter status=exited --filter name=runner-": []byte("abc123\n"),
			"inspect": []byte(old),
			"/bin/docker volume ls -q --filter dangling=true": []byte("runner-vol\n"),
		},
	}
	var out bytes.Buffer
	now := time.Now()
	ctrl, err := Config{
		Exec:  exec,
		Out:   &out,
		Clock: func() time.Time { return now },
	}.Create()
	if err != nil {
		t.Fatal(err)
	}
	if err := ctrl.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Removed stopped Runner container") {
		t.Fatalf("out: %q", out.String())
	}
	if !strings.Contains(out.String(), "Removed runner-vol") {
		t.Fatalf("out: %q", out.String())
	}
}

func TestRun_keepsYoungContainer(t *testing.T) {
	young := time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339Nano)
	exec := &dockerExec{
		dockerBin: "/bin/docker",
		responses: map[string][]byte{
			"/bin/docker info": nil,
			"/bin/docker ps -aq --filter status=exited --filter name=runner-": []byte("abc123\n"),
			"inspect": []byte(young),
			"/bin/docker volume ls -q --filter dangling=true": nil,
		},
	}
	var out bytes.Buffer
	ctrl, err := Config{Exec: exec, Out: &out}.Create()
	if err != nil {
		t.Fatal(err)
	}
	if err := ctrl.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Removed stopped Runner container") {
		t.Fatalf("should not remove young container: %q", out.String())
	}
}

var _ gitexec.Exec = (*dockerExec)(nil)
