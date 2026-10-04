package runners

import (
	"testing"

	"github.com/behaviorengineering/runnerconcierge/pkg/inventory"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

func TestJoinTargets_brewLaunchdDedupe(t *testing.T) {
	rep := &inventory.Report{
		Configs: []inventory.ConfigReport{{
			Path:     "/cfg.toml",
			Readable: true,
			Runners:  []inventory.RunnerEntry{{Name: "host", URL: "https://gitlab.com", Executor: "shell"}},
		}},
		Services: []service.Ownership{
			{ServiceName: "gitlab-runner", Kind: "brew_services", State: "started", ConfigPath: "/cfg.toml"},
			{ServiceName: "sh.brew.gitlab-runner", Kind: "launchd", State: "started", ConfigPath: "/cfg.toml"},
		},
	}
	targets := JoinTargets(rep)
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].ServiceName != "gitlab-runner" {
		t.Fatalf("service %q", targets[0].ServiceName)
	}
}

func TestJoinTargets_twoRunnersOneService(t *testing.T) {
	rep := &inventory.Report{
		Configs: []inventory.ConfigReport{{
			Path:     "/cfg.toml",
			Readable: true,
			Runners: []inventory.RunnerEntry{
				{Name: "a", URL: "https://gitlab.com"},
				{Name: "b", URL: "https://gitlab.com"},
			},
		}},
		Services: []service.Ownership{
			{ServiceName: "gitlab-runner", Kind: "brew_services", ConfigPath: "/cfg.toml"},
		},
	}
	targets := JoinTargets(rep)
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}
	if targets[0].EntryCount != 2 || targets[1].EntryCount != 2 {
		t.Fatalf("entry counts %d %d", targets[0].EntryCount, targets[1].EntryCount)
	}
}

func TestJoinTargets_orphanService(t *testing.T) {
	rep := &inventory.Report{
		Services: []service.Ownership{
			{
				ServiceName: "orphan",
				Kind:        "launchd",
				State:       "stopped",
				ConfigPath:  "/x.toml",
				Role:        service.RoleHelper,
				MatchReason: service.MatchReasonLaunchdLabel,
			},
		},
	}
	targets := JoinTargets(rep)
	if len(targets) != 1 || targets[0].Key != "svc:orphan" {
		t.Fatalf("got %+v", targets)
	}
	if targets[0].Kind != TargetKindServiceOnly || targets[0].Role != service.RoleHelper {
		t.Fatalf("kind/role %+v", targets[0])
	}
}

func TestJoinTargets_registrationsAndServices(t *testing.T) {
	rep := &inventory.Report{
		Configs: []inventory.ConfigReport{{
			Path:     "/cfg.toml",
			Readable: true,
			Runners: []inventory.RunnerEntry{
				{Name: "a"},
				{Name: "b"},
			},
		}},
		Services: []service.Ownership{
			{ServiceName: "gitlab-runner", Kind: "brew_services", State: "started", Role: service.RoleSupervisor, MatchReason: service.MatchReasonBrewFormula, ProcessUp: true},
			{ServiceName: "com.hector.gitlab-runner-docker-cleanup", Kind: "launchd", State: "running", Role: service.RoleHelper, MatchReason: service.MatchReasonLaunchdLabel, ProcessUp: true},
		},
	}
	targets := JoinTargets(rep)
	if len(targets) != 4 {
		t.Fatalf("expected 4 targets, got %d", len(targets))
	}
	if targets[0].Kind != TargetKindRegistration {
		t.Fatalf("first should be registration")
	}
	if targets[2].Role != service.RoleSupervisor {
		t.Fatalf("supervisor at index 2: %+v", targets[2])
	}
}

func TestPickTarget_ambiguous(t *testing.T) {
	targets := []Target{
		{Name: "host", ServiceName: "svc-a", ConfigPath: "/a.toml"},
		{Name: "host", ServiceName: "svc-b", ConfigPath: "/b.toml"},
	}
	_, err := PickTarget(targets, "host", "", "")
	if err == nil {
		t.Fatal("expected ambiguity error")
	}
}

func TestPickTarget_uniqueName(t *testing.T) {
	targets := []Target{
		{Name: "alpha", ServiceName: "svc-a"},
		{Name: "beta", ServiceName: "svc-b"},
	}
	tgt, err := PickTarget(targets, "beta", "", "")
	if err != nil || tgt.Name != "beta" {
		t.Fatalf("pick: %v %+v", err, tgt)
	}
}
