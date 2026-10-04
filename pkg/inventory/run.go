package inventory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/detect"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

// Run collects a machine inventory report.
func Run(ctx context.Context, opts Options) (*Report, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errdefs.New("inventory.Run", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if opts.Exec == nil {
		return nil, fmt.Errorf("inventory: exec is nil")
	}
	login := strings.TrimSpace(opts.LoginUser)
	if login == "" {
		u, err := service.LoginUser(opts.Exec)
		if err != nil {
			return nil, err
		}
		login = u
	}
	rep := &Report{
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
		LoginUser: login,
		Elevated:  isElevated(),
	}
	paths := discoverPaths(opts)
	svcMgr := service.New(opts.Exec)
	services, err := svcMgr.ListOwnership(ctx)
	if err != nil && !isUnsupportedOS(err) {
		return nil, err
	}
	rep.Services = services
	for _, p := range paths {
		if p == "" {
			continue
		}
		cr := ConfigReport{Path: p, SystemPath: service.IsSystemConfigPath(p)}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		runners, err := parseConfigFile(p)
		if err != nil {
			cr.Readable = false
			rep.Findings = append(rep.Findings, classifyConfigPath(p, false, rep.Elevated)...)
			rep.Configs = append(rep.Configs, cr)
			continue
		}
		cr.Readable = true
		cr.Runners = runners
		rep.Configs = append(rep.Configs, cr)
		rep.Findings = append(rep.Findings, classifyConfigPath(p, true, rep.Elevated)...)
	}
	bin, lookErr := opts.Exec.LookPath("gitlab-runner")
	if lookErr != nil {
		rep.Findings = append(rep.Findings, Finding{
			Code:    errdefs.CodeRunnerBinaryMissing,
			Block:   true,
			Message: "gitlab-runner is not on PATH",
		})
	} else {
		rep.RunnerBinary = bin
		if ver, err := opts.Exec.Run(ctx, bin, "--version"); err == nil {
			rep.RunnerVer = strings.TrimSpace(string(ver))
		}
	}
	for _, svc := range services {
		cfg := svc.ConfigPath
		if cfg == "" && len(rep.Configs) == 1 {
			cfg = rep.Configs[0].Path
		}
		rep.Findings = append(rep.Findings, classifyOwnership(login, svc, cfg)...)
	}
	if len(rep.Configs) > 0 && len(services) == 0 {
		st, stErr := svcMgr.Status(ctx)
		if stErr != nil || strings.Contains(strings.ToLower(st), "not installed") {
			rep.Findings = append(rep.Findings, Finding{
				Code:    errdefs.CodeServiceMissing,
				Block:   false,
				Message: "runner config found but no gitlab-runner service detected",
			})
		}
	}
	doc := detect.NewDoctor(opts.Exec)
	dctx, cancel := detect.WithDoctorDeadline(ctx)
	drep, _ := doc.Run(dctx, false)
	cancel()
	if drep != nil {
		for _, iss := range drep.Issues {
			if iss.Code == errdefs.CodeProcessConflict {
				rep.Findings = append(rep.Findings, Finding{
					Code:    iss.Code,
					Block:   iss.Block,
					Message: iss.Message,
				})
			}
		}
	}
	rep.NextActions = buildNextActions(rep.Findings)
	return rep, nil
}

func discoverPaths(opts Options) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, p := range service.CandidateConfigPaths() {
		add(p)
	}
	for _, p := range opts.ExtraConfigPaths {
		add(p)
	}
	return out
}

func buildNextActions(findings []Finding) []string {
	var actions []string
	for _, f := range findings {
		if !f.Repairable {
			continue
		}
		switch f.Code {
		case errdefs.CodeBadServiceLogon, errdefs.CodeWrongServiceUser, errdefs.CodeSystemConfig:
			actions = append(actions, "runnerconcierge repair-service --runner-config "+f.ConfigPath)
		case errdefs.CodeElevationRequired, errdefs.CodeConfigUnreadable:
			actions = append(actions, "re-run in an elevated/administrator session, then runnerconcierge status")
		}
	}
	return dedupeStrings(actions)
}

func dedupeStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func isUnsupportedOS(err error) bool {
	var de *errdefs.Error
	if errors.As(err, &de) {
		return de.Code == errdefs.CodeUnsupportedOS
	}
	return false
}
