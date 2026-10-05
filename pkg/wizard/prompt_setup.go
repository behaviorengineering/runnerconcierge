package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/gitlabrunner"
	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

const (
	runnerTypeProject = "project_type"
	runnerTypeGroup   = "group_type"
)

func (r *Runner) promptSetupOptions(ctx context.Context, pr prompt.Prompter, cp *state.Checkpoint) error {
	if r == nil || r.opts.NonInteractive {
		return nil
	}
	if pr == nil {
		return nil
	}
	skipScope := strings.TrimSpace(r.opts.RunnerToken) != ""

	if !skipScope && r.effectiveRunnerType() == "" {
		idx, err := pr.Select(ctx, "Register runner for", []string{
			"Project (one GitLab project)",
			"Group (all projects in a group)",
		})
		if err != nil {
			return err
		}
		switch idx {
		case 0:
			r.opts.RunnerType = runnerTypeProject
		case 1:
			r.opts.RunnerType = runnerTypeGroup
		default:
			return fmt.Errorf("wizard: invalid runner scope selection")
		}
	}

	if !skipScope {
		switch r.effectiveRunnerType() {
		case runnerTypeGroup:
			if strings.TrimSpace(r.opts.GroupPath) == "" && r.preset.GroupID <= 0 {
				client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)
				if err := r.promptGroup(ctx, pr, client); err != nil {
					return err
				}
			}
		case runnerTypeProject:
			if strings.TrimSpace(r.opts.ProjectPath) == "" && strings.TrimSpace(r.preset.RepoPath) == "" {
				client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)
				if err := r.promptProject(ctx, pr, client); err != nil {
					return err
				}
			}
		}
	}

	defaultTags := strings.Join(r.opts.TagList, ",")
	if defaultTags == "" && len(r.preset.TagList) > 0 {
		defaultTags = strings.Join(r.preset.TagList, ",")
	}
	placeholder := "e.g. mac-ci,docker"
	val, err := pr.Input(ctx, "Runner tags (comma-separated)", placeholder, defaultTags)
	if err != nil {
		return err
	}
	val = strings.TrimSpace(val)
	if val != "" {
		r.opts.TagList = splitCommaTags(val)
	} else if defaultTags != "" {
		r.opts.TagList = splitCommaTags(defaultTags)
	}

	if strings.TrimSpace(r.opts.Executor) == "" && strings.TrimSpace(r.preset.Executor) == "" {
		opts := []string{"docker", "shell"}
		defaultExec := strings.TrimSpace(r.cfg.DefaultExecutor)
		title := "Executor for CI jobs"
		if defaultExec == "docker" || defaultExec == "shell" {
			title = fmt.Sprintf("Executor for CI jobs (default: %s)", defaultExec)
		}
		idx, err := pr.Select(ctx, title, opts)
		if err != nil {
			return err
		}
		if idx >= 0 && idx < len(opts) {
			r.opts.Executor = opts[idx]
		}
	}

	if cp != nil {
		r.syncSetupCheckpoint(cp)
		return r.saveCheckpoint(cp)
	}
	return nil
}

func (r *Runner) promptProject(ctx context.Context, pr prompt.Prompter, client *gitlabrunner.Client) error {
	hint, _ := gitlabrunner.ProjectPathFromGitRemote(ctx, r.exec, r.gitlabURL())
	projects, listErr := client.ListMemberProjects(ctx)

	var paths []string
	seen := map[string]bool{}
	if hint != "" {
		paths = append(paths, hint)
		seen[hint] = true
	}
	if listErr == nil {
		for _, p := range projects {
			path := strings.TrimSpace(p.PathWithNamespace)
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			paths = append(paths, path)
		}
	}

	if len(paths) > 0 {
		labels := make([]string, len(paths))
		for i, p := range paths {
			if i == 0 && hint != "" && p == hint {
				labels[i] = p + " (from git remote)"
			} else {
				labels[i] = p
			}
		}
		idx, err := pr.Select(ctx, "Select GitLab project", labels)
		if err != nil {
			return err
		}
		if idx < 0 || idx >= len(paths) {
			return fmt.Errorf("wizard: invalid project selection")
		}
		r.opts.ProjectPath = paths[idx]
		return nil
	}

	placeholder := "group/project"
	defaultVal := strings.TrimSpace(r.opts.ProjectPath)
	if defaultVal == "" && hint != "" {
		placeholder = hint
		defaultVal = hint
	}
	val, err := pr.Input(ctx, "GitLab project path", placeholder, defaultVal)
	if err != nil {
		return err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return fmt.Errorf("wizard: project path is required")
	}
	r.opts.ProjectPath = val
	return nil
}

func (r *Runner) promptGroup(ctx context.Context, pr prompt.Prompter, client *gitlabrunner.Client) error {
	groups, listErr := client.ListMemberGroups(ctx)
	if listErr != nil && r.out != nil {
		r.out("could not list GitLab groups: " + listErr.Error())
	}

	var paths []string
	seen := map[string]bool{}
	if listErr == nil {
		for _, g := range groups {
			path := strings.TrimSpace(g.FullPath)
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			paths = append(paths, path)
		}
	}

	if len(paths) > 0 {
		labels := make([]string, len(paths))
		for i, p := range paths {
			labels[i] = p
		}
		idx, err := pr.Select(ctx, "Select GitLab group", labels)
		if err != nil {
			return err
		}
		if idx < 0 || idx >= len(paths) {
			return fmt.Errorf("wizard: invalid group selection")
		}
		r.opts.GroupPath = paths[idx]
		return nil
	}

	defaultVal := strings.TrimSpace(r.opts.GroupPath)
	val, err := pr.Input(ctx, "GitLab group path", "my-group", defaultVal)
	if err != nil {
		return err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return fmt.Errorf("wizard: group path is required")
	}
	r.opts.GroupPath = val
	return nil
}

func splitCommaTags(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
