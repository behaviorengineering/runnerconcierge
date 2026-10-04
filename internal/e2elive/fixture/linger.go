//go:build e2e_live && (darwin || windows)

package fixture

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

type configSnapshot map[string][]byte

func snapshotCandidateConfigs() (configSnapshot, error) {
	out := configSnapshot{}
	for _, p := range service.CandidateConfigPaths() {
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		out[p] = data
	}
	return out, nil
}

func verifyConfigSnapshots(before configSnapshot) error {
	for p, want := range before {
		got, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				return wrap("verifyConfigSnapshots", errdefs.CodeProcessConflict, "candidate config removed: "+p, err)
			}
			return err
		}
		if string(got) != string(want) {
			return wrap("verifyConfigSnapshots", errdefs.CodeProcessConflict, "candidate config mutated: "+p, nil)
		}
	}
	for _, p := range service.CandidateConfigPaths() {
		if _, ok := before[p]; ok {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return wrap("verifyConfigSnapshots", errdefs.CodeProcessConflict, "candidate config created: "+p, nil)
		}
	}
	return nil
}

func assertLingerServices(ctx context.Context, exec gitexec.Exec, initialCount int, fixtureServiceName string) error {
	svc := service.New(exec)
	list, err := svc.ListOwnership(ctx)
	if err != nil {
		return err
	}
	for _, o := range list {
		name := o.ServiceName
		if name == fixtureServiceName || strings.HasPrefix(name, namePrefix) {
			return wrap("linger", errdefs.CodeProcessConflict,
				"fixture gitlab-runner service still present after dispose: "+name, nil)
		}
	}
	if initialCount > 0 || allowExistingServices() {
		homelab := 0
		for _, o := range list {
			if strings.HasPrefix(o.ServiceName, namePrefix) {
				continue
			}
			homelab++
		}
		if homelab != initialCount {
			return wrap("linger", errdefs.CodeProcessConflict,
				fmt.Sprintf("gitlab-runner service count %d after dispose, want %d", homelab, initialCount), nil)
		}
		return nil
	}
	if len(list) > 0 {
		return wrap("linger", errdefs.CodeProcessConflict, "gitlab-runner service still present after dispose", nil)
	}
	return nil
}

func countRunnerServices(ctx context.Context, exec gitexec.Exec) (int, error) {
	svc := service.New(exec)
	list, err := svc.ListOwnership(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, o := range list {
		if strings.HasPrefix(o.ServiceName, namePrefix) {
			continue
		}
		n++
	}
	return n, nil
}

func safetyAbortExisting(ctx context.Context, exec gitexec.Exec, serviceName string) error {
	svc := service.New(exec)
	list, err := svc.ListOwnership(ctx)
	if err != nil {
		return err
	}
	for _, o := range list {
		if o.ServiceName == serviceName {
			return wrap("safetyAbortExisting", errdefs.CodeProcessConflict,
				"fixture service "+serviceName+" already present; remove it or pick a new run", nil)
		}
	}
	return nil
}
