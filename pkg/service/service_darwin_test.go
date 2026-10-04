//go:build darwin

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

type scriptedExec struct {
	lookPath map[string]error
	run      map[string]struct {
		out []byte
		err error
	}
}

func (s *scriptedExec) LookPath(name string) (string, error) {
	if err, ok := s.lookPath[name]; ok {
		if err != nil {
			return "", err
		}
		return name, nil
	}
	return name, nil
}

func (s *scriptedExec) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	got, ok := s.run[key]
	if !ok {
		return nil, fmt.Errorf("unexpected exec: %s", key)
	}
	return got.out, got.err
}

func (s *scriptedExec) RunJSON(ctx context.Context, name string, args ...string) ([]byte, error) {
	return s.Run(ctx, name, args...)
}

var _ gitexec.Exec = (*scriptedExec)(nil)

func TestDarwinStatusUsesBrewWhenGitLabNativeMissing(t *testing.T) {
	m := &darwinManager{exec: &scriptedExec{
		run: map[string]struct {
			out []byte
			err error
		}{
			"gitlab-runner status": {
				err: fmt.Errorf("gitlab-runner status: gitlab-runner: the service is not installed: exit status 1"),
			},
			"brew services list": {
				out: []byte("Name          Status User   File\ngitlab-runner started         hector ~/Library/LaunchAgents/sh.brew.gitlab-runner.plist\n"),
			},
		},
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st, err := m.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if st != "gitlab-runner brew_services started user=hector" {
		t.Fatalf("got %q", st)
	}
}

func TestDarwinStatusMissingBoth(t *testing.T) {
	m := &darwinManager{exec: &scriptedExec{
		lookPath: map[string]error{"brew": errors.New("not found")},
		run: map[string]struct {
			out []byte
			err error
		}{
			"gitlab-runner status": {
				err: fmt.Errorf("gitlab-runner status: the service is not installed: exit status 1"),
			},
		},
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := m.Status(ctx)
	if err == nil {
		t.Fatal("expected missing")
	}
	if errdefs.CodeOf(err) != errdefs.CodeServiceMissing {
		t.Fatalf("code %s", errdefs.CodeOf(err))
	}
	if strings.Contains(err.Error(), "gitlab-runner status:") {
		t.Fatalf("argv dump in Error(): %s", err.Error())
	}
}
