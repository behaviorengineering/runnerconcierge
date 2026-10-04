package runners

import (
	"fmt"
	"sort"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/inventory"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

// TargetKind classifies a picker row.
type TargetKind string

const (
	TargetKindRegistration TargetKind = "registration"
	TargetKindServiceOnly  TargetKind = "service_only"
)

// Target is one selectable runner on this machine.
type Target struct {
	Key          string     `json:"key"`
	Kind         TargetKind `json:"target_kind"`
	Name         string     `json:"name"`
	URL          string     `json:"url"`
	Executor     string     `json:"executor"`
	GitLabID     int        `json:"gitlab_id"`
	ConfigPath   string     `json:"config_path"`
	ServiceName  string     `json:"service_name"`
	ServiceKind  string     `json:"service_kind"`
	ServiceState string     `json:"service_state"`
	UseBrew      bool       `json:"use_brew"`
	EntryCount   int        `json:"entry_count"`
	UnitPath     string     `json:"unit_path"`
	Command      string     `json:"command"`
	MatchReason  string     `json:"match_reason"`
	Role         string     `json:"role"`
	ProcessUp    bool       `json:"process_up"`
	LogonUser    string     `json:"logon_user"`
}

// JoinTargets builds picker rows from an inventory report.
func JoinTargets(rep *inventory.Report) []Target {
	if rep == nil {
		return nil
	}
	services := dedupeServices(rep.Services)
	configCount := 0
	for _, c := range rep.Configs {
		if c.Readable {
			configCount++
		}
	}
	attached := map[string]bool{}
	var out []Target
	for _, cfg := range rep.Configs {
		if !cfg.Readable {
			continue
		}
		count := len(cfg.Runners)
		for _, r := range cfg.Runners {
			svc := attachService(services, cfg.Path, configCount, attached)
			t := Target{
				Key:        targetKey(cfg.Path, r.Name),
				Kind:       TargetKindRegistration,
				Name:       strings.TrimSpace(r.Name),
				URL:        strings.TrimSpace(r.URL),
				Executor:   strings.TrimSpace(r.Executor),
				GitLabID:   r.GitLabID,
				ConfigPath: cfg.Path,
				EntryCount: count,
			}
			if svc != nil {
				applyServiceFields(&t, svc)
				attached[svc.ServiceName] = true
			}
			out = append(out, t)
		}
	}
	for i := range services {
		if attached[services[i].ServiceName] {
			continue
		}
		t := Target{
			Key:         "svc:" + services[i].ServiceName,
			Kind:        TargetKindServiceOnly,
			ConfigPath:  services[i].ConfigPath,
			ServiceName: services[i].ServiceName,
		}
		applyServiceFields(&t, &services[i])
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind == TargetKindRegistration
		}
		if roleRank(out[i].Role) != roleRank(out[j].Role) {
			return roleRank(out[i].Role) < roleRank(out[j].Role)
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		if out[i].ConfigPath != out[j].ConfigPath {
			return out[i].ConfigPath < out[j].ConfigPath
		}
		return out[i].ServiceName < out[j].ServiceName
	})
	return out
}

func roleRank(role string) int {
	switch role {
	case service.RoleSupervisor:
		return 0
	case service.RoleHelper:
		return 1
	case service.RoleFixture:
		return 2
	default:
		return 3
	}
}

func applyServiceFields(t *Target, svc *service.Ownership) {
	if t == nil || svc == nil {
		return
	}
	t.ServiceName = svc.ServiceName
	t.ServiceKind = svc.Kind
	t.ServiceState = svc.State
	t.UseBrew = svc.Kind == "brew_services" || svc.ServiceName == "sh.brew.gitlab-runner"
	t.UnitPath = svc.UnitPath
	t.Command = svc.Command
	t.MatchReason = svc.MatchReason
	t.Role = svc.Role
	t.ProcessUp = svc.ProcessUp
	t.LogonUser = svc.LogonUser
	if strings.TrimSpace(t.ConfigPath) == "" {
		t.ConfigPath = svc.ConfigPath
	}
}

func targetKey(configPath, name string) string {
	if strings.TrimSpace(name) == "" {
		return configPath
	}
	return configPath + "|" + strings.TrimSpace(name)
}

func dedupeServices(in []service.Ownership) []service.Ownership {
	var brew, brewPlist *service.Ownership
	var rest []service.Ownership
	for i := range in {
		s := in[i]
		if s.Kind == "brew_services" && s.ServiceName == "gitlab-runner" {
			brew = &in[i]
			continue
		}
		if s.Kind == "launchd" && s.ServiceName == "sh.brew.gitlab-runner" {
			brewPlist = &in[i]
			continue
		}
		rest = append(rest, s)
	}
	if brew != nil {
		rest = append(rest, *brew)
	} else if brewPlist != nil {
		rest = append(rest, *brewPlist)
	}
	return rest
}

func attachService(services []service.Ownership, configPath string, configCount int, attached map[string]bool) *service.Ownership {
	for i := range services {
		s := &services[i]
		if attached[s.ServiceName] {
			continue
		}
		if s.ConfigPath == configPath {
			return s
		}
	}
	if len(services) == 1 && configCount == 1 {
		for i := range services {
			if !attached[services[i].ServiceName] {
				return &services[i]
			}
		}
	}
	return nil
}

// PickTarget resolves a target from flags.
func PickTarget(targets []Target, name, serviceName, configPath string) (*Target, error) {
	name = strings.TrimSpace(name)
	serviceName = strings.TrimSpace(serviceName)
	configPath = strings.TrimSpace(configPath)
	var matches []Target
	for _, t := range targets {
		if name != "" && !strings.EqualFold(t.Name, name) {
			continue
		}
		if serviceName != "" && t.ServiceName != serviceName {
			continue
		}
		if configPath != "" && t.ConfigPath != configPath {
			continue
		}
		matches = append(matches, t)
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("runners: no runner matched selection")
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("runners: selection is ambiguous; pass --service or --config")
	}
}

// IsHelperServiceTarget reports whether actions should refer to a service unit, not a registered runner.
func IsHelperServiceTarget(t *Target) bool {
	if t == nil {
		return false
	}
	return t.Kind == TargetKindServiceOnly || t.Role == service.RoleHelper
}
