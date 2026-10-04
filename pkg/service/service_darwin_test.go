//go:build darwin

package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

type argvExec struct {
	last []string
}

func (a *argvExec) LookPath(name string) (string, error) { return name, nil }
func (a *argvExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	a.last = append([]string{name}, args...)
	return nil, nil
}
func (a *argvExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return a.Run(ctx, name, args...)
}

func TestDarwinStop_useBrew(t *testing.T) {
	exec := &argvExec{}
	m := newPlatformManager(exec).(*darwinManager)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.Stop(ctx, StopOpts{UseBrew: true}); err != nil {
		t.Fatal(err)
	}
	if exec.last[0] != "brew" || !strings.Contains(strings.Join(exec.last, " "), "stop") {
		t.Fatalf("argv %v", exec.last)
	}
}

func TestDarwinStop_namedService(t *testing.T) {
	exec := &argvExec{}
	m := newPlatformManager(exec).(*darwinManager)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.Stop(ctx, StopOpts{ServiceName: "sh.brew.gitlab-runner"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(exec.last, " "), "--service") {
		t.Fatalf("argv %v", exec.last)
	}
}
