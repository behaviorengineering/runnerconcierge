package runners

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/gitlabrunner"
	"github.com/behaviorengineering/runnerconcierge/pkg/inventory"
	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
	"github.com/behaviorengineering/runnerconcierge/pkg/repair"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

// Run lists runners and performs the selected action.
func (c *Controller) Run(ctx context.Context) error {
	if c == nil {
		return errdefs.New("runners.gitlab", errdefs.CodeCreateFailed, "controller is nil", nil)
	}
	if _, ok := ctx.Deadline(); !ok {
		return errdefs.New("runners.gitlab", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	extra := []string{}
	if strings.TrimSpace(c.cfg.ConfigPath) != "" {
		extra = append(extra, strings.TrimSpace(c.cfg.ConfigPath))
	}
	rep, err := inventory.Run(ctx, inventory.Options{Exec: c.cfg.Exec, ExtraConfigPaths: extra})
	if err != nil {
		return errdefs.New("runners.gitlab", errdefs.CodeOf(err), "could not collect inventory", err)
	}
	targets := JoinTargets(rep)
	picked := strings.TrimSpace(c.cfg.Name) != "" || strings.TrimSpace(c.cfg.ServiceName) != "" || strings.TrimSpace(c.cfg.ConfigPath) != ""
	if c.cfg.JSON && !picked {
		enc := json.NewEncoder(c.cfg.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(targets)
	}
	if c.cfg.JSON && picked {
		t, err := PickTarget(targets, c.cfg.Name, c.cfg.ServiceName, c.cfg.ConfigPath)
		if err != nil {
			return errdefs.New("runners.gitlab", errdefs.CodeInvalidScope, err.Error(), nil)
		}
		enc := json.NewEncoder(c.cfg.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(t)
	}
	if len(targets) == 0 {
		_ = renderList(c.cfg.Out, targets)
		return nil
	}
	t, err := c.pickTarget(ctx, targets)
	if err != nil {
		return err
	}
	act := c.cfg.Action
	if act == "" {
		act, err = c.pickAction(ctx)
		if err != nil {
			return err
		}
	}
	return c.dispatch(ctx, t, act)
}

func (c *Controller) pickTarget(ctx context.Context, targets []Target) (*Target, error) {
	if strings.TrimSpace(c.cfg.Name) != "" || strings.TrimSpace(c.cfg.ServiceName) != "" {
		t, err := PickTarget(targets, c.cfg.Name, c.cfg.ServiceName, c.cfg.ConfigPath)
		if err != nil {
			return nil, errdefs.New("runners.gitlab", errdefs.CodeInvalidScope, err.Error(), nil)
		}
		return t, nil
	}
	if c.cfg.NonInteractive {
		return nil, errdefs.New("runners.gitlab", errdefs.CodeInvalidScope, "pass --name or --service in non-interactive mode", nil)
	}
	pr := c.cfg.Prompter
	if pr == nil {
		pr = prompt.ForTTY()
	}
	var labels []string
	for _, t := range targets {
		labels = append(labels, fmt.Sprintf("%s  service=%s  config=%s",
			labelOrDash(t.Name), labelOrDash(t.ServiceName), labelOrDash(t.ConfigPath)))
	}
	idx, err := pr.Select(ctx, "Select a runner", labels)
	if err != nil {
		return nil, err
	}
	return &targets[idx], nil
}

func (c *Controller) pickAction(ctx context.Context) (Action, error) {
	if c.cfg.NonInteractive {
		return "", errdefs.New("runners.gitlab", errdefs.CodeInvalidScope, "pass --action in non-interactive mode", nil)
	}
	pr := c.cfg.Prompter
	if pr == nil {
		pr = prompt.ForTTY()
	}
	opts := []string{
		"Inspect",
		"Stop service",
		"Start service",
		"Repair service user",
		"Remove runner",
	}
	idx, err := pr.Select(ctx, "Choose an action", opts)
	if err != nil {
		return "", err
	}
	switch idx {
	case 0:
		return ActionInspect, nil
	case 1:
		return ActionStop, nil
	case 2:
		return ActionStart, nil
	case 3:
		return ActionRepair, nil
	case 4:
		return ActionRemove, nil
	default:
		return "", fmt.Errorf("runners: invalid action index")
	}
}

func (c *Controller) dispatch(ctx context.Context, t *Target, act Action) error {
	switch act {
	case ActionInspect:
		return c.actionInspect(ctx, t)
	case ActionStop:
		return c.actionStop(ctx, t)
	case ActionStart:
		return c.actionStart(ctx, t)
	case ActionRepair:
		return c.actionRepair(ctx, t)
	case ActionRemove:
		return c.actionRemove(ctx, t)
	default:
		return errdefs.New("runners.gitlab", errdefs.CodeInvalidScope, "unknown action", nil)
	}
}

func (c *Controller) actionInspect(ctx context.Context, t *Target) error {
	online := ""
	if !c.cfg.LocalOnly {
		id, _ := c.resolveGitLabID(ctx, t)
		if id > 0 {
			online = c.runnerOnline(ctx, t, id)
		}
	}
	return renderTarget(c.cfg.Out, t, online)
}

func (c *Controller) actionStop(ctx context.Context, t *Target) error {
	if err := c.guardSupervisorOnlyTarget(t, "stop"); err != nil {
		return err
	}
	if err := c.warnServiceScope(ctx, t); err != nil {
		return err
	}
	if strings.TrimSpace(t.ServiceName) == "" {
		return errdefs.New("runners.gitlab.stop", errdefs.CodeServiceMissing, "no service linked to this runner", nil)
	}
	return c.cfg.Service.Stop(ctx, service.StopOpts{ServiceName: t.ServiceName, UseBrew: t.UseBrew})
}

func (c *Controller) actionStart(ctx context.Context, t *Target) error {
	if err := c.guardSupervisorOnlyTarget(t, "start"); err != nil {
		return err
	}
	if err := c.warnServiceScope(ctx, t); err != nil {
		return err
	}
	if strings.TrimSpace(t.ServiceName) == "" {
		return errdefs.New("runners.gitlab.start", errdefs.CodeServiceMissing, "no service linked to this runner", nil)
	}
	return c.cfg.Service.Start(ctx, service.StartOpts{ServiceName: t.ServiceName, UseBrew: t.UseBrew})
}

func (c *Controller) actionRepair(ctx context.Context, t *Target) error {
	if strings.TrimSpace(t.ConfigPath) == "" {
		return errdefs.New("runners.gitlab.repair", errdefs.CodeInvalidScope, "config path is required for repair", nil)
	}
	if err := c.warnServiceScope(ctx, t); err != nil {
		return err
	}
	pr := c.cfg.Prompter
	if pr == nil && !c.cfg.AllowYes {
		pr = prompt.ForTTY()
	}
	res, err := repair.Run(ctx, repair.Options{
		Exec:             c.cfg.Exec,
		Prompter:         pr,
		ConfigPath:       t.ConfigPath,
		ServiceName:      t.ServiceName,
		WindowsPassword:  c.cfg.WindowsPassword,
		AllowDestructive: c.cfg.AllowYes,
		UseBrewServices:  t.UseBrew,
	})
	if err != nil {
		return errdefs.New("runners.gitlab.repair", errdefs.CodeOf(err), "repair failed", err)
	}
	repair.PrintResult(c.cfg.Out, res)
	return nil
}

func (c *Controller) actionRemove(ctx context.Context, t *Target) error {
	if err := c.guardSupervisorOnlyTarget(t, "remove"); err != nil {
		return err
	}
	if !c.cfg.AllowYes {
		pr := c.cfg.Prompter
		if pr == nil {
			if c.cfg.NonInteractive {
				return errdefs.New("runners.gitlab.remove", errdefs.CodeInvalidScope, "pass --yes to remove", nil)
			}
			pr = prompt.ForTTY()
		}
		msg := removeConfirmMessage(t)
		ok, err := pr.Confirm(ctx, msg)
		if err != nil {
			return err
		}
		if !ok {
			return errdefs.New("runners.gitlab.remove", errdefs.CodeInvalidScope, "cancelled", nil)
		}
	}
	var resolveErr error
	gitlabID := c.cfg.GitLabID
	if gitlabID <= 0 {
		gitlabID = t.GitLabID
	}
	if gitlabID <= 0 && !c.cfg.LocalOnly {
		gitlabID, resolveErr = c.resolveGitLabID(ctx, t)
	}
	if strings.TrimSpace(t.Name) != "" && strings.TrimSpace(t.ConfigPath) != "" {
		backup, err := backupConfig(t.ConfigPath)
		if err != nil {
			return errdefs.New("runners.gitlab.remove", errdefs.CodeCreateFailed, "could not backup config", err)
		}
		if err := c.unregisterLocal(ctx, t); err != nil && !unregisterBenign(err) {
			return errdefs.New("runners.gitlab.remove", errdefs.CodeRegisterFailed,
				fmt.Sprintf("unregister failed (backup at %s)", backup), err)
		}
	} else if strings.TrimSpace(t.ServiceName) != "" {
		// Orphan service only.
	} else {
		return errdefs.New("runners.gitlab.remove", errdefs.CodeInvalidScope, "nothing to remove for this target", nil)
	}
	if t.EntryCount <= 1 && strings.TrimSpace(t.ServiceName) != "" {
		bin, _ := c.cfg.Exec.LookPath("gitlab-runner")
		if err := c.cfg.Service.Uninstall(ctx, service.UninstallOpts{
			BinaryPath:  bin,
			ConfigPath:  t.ConfigPath,
			ServiceName: t.ServiceName,
			UseBrew:     t.UseBrew,
		}); err != nil {
			return errdefs.New("runners.gitlab.remove", errdefs.CodeOf(err), "service uninstall failed", err)
		}
	}
	if c.cfg.LocalOnly {
		_, _ = fmt.Fprintln(c.cfg.Out, "local remove complete")
		return nil
	}
	if gitlabID > 0 {
		client := c.clientForTarget(t)
		pat := c.pat()
		if err := client.DeleteRunner(ctx, gitlabID, pat); err != nil {
			_, _ = fmt.Fprintln(c.cfg.Out, "local remove complete; GitLab delete failed")
			return errdefs.New("runners.gitlab.remove", errdefs.CodeOf(err), "could not delete GitLab runner", err)
		}
		_, _ = fmt.Fprintln(c.cfg.Out, "remove complete")
		return nil
	}
	_, _ = fmt.Fprintln(c.cfg.Out, "local remove complete")
	if resolveErr != nil {
		return errdefs.New("runners.gitlab.remove", errdefs.CodeOf(resolveErr), "GitLab runner id unresolved; delete in GitLab UI", resolveErr)
	}
	return errdefs.New("runners.gitlab.remove", errdefs.CodeInvalidScope, "GitLab runner id unresolved; delete in GitLab UI", nil)
}

func (c *Controller) unregisterLocal(ctx context.Context, t *Target) error {
	bin, err := c.cfg.Exec.LookPath("gitlab-runner")
	if err != nil {
		return errdefs.New("runners.gitlab.remove", errdefs.CodeRunnerBinaryMissing, "gitlab-runner not on PATH", err)
	}
	args, err := gitlabrunner.BuildUnregisterArgv(gitlabrunner.UnregisterArgs{
		Name:       t.Name,
		URL:        t.URL,
		ConfigPath: t.ConfigPath,
	})
	if err != nil {
		return err
	}
	client := c.clientForTarget(t)
	return client.Unregister(ctx, bin, args)
}

func (c *Controller) resolveGitLabID(ctx context.Context, t *Target) (int, error) {
	if c.cfg.GitLabID > 0 {
		return c.cfg.GitLabID, nil
	}
	if t.GitLabID > 0 {
		return t.GitLabID, nil
	}
	client := c.clientForTarget(t)
	return client.MatchRunner(ctx, c.pat(), t.Name, t.URL)
}

func (c *Controller) clientForTarget(t *Target) *gitlabrunner.Client {
	base := strings.TrimSpace(t.URL)
	if base == "" {
		base = c.cfg.Client.BaseURL
	}
	return gitlabrunner.NewClient(base, c.cfg.Exec, nil)
}

func (c *Controller) pat() string {
	if strings.TrimSpace(c.cfg.PAT) != "" {
		return strings.TrimSpace(c.cfg.PAT)
	}
	p, err := config.ResolvePAT()
	if err != nil {
		return ""
	}
	return p
}

func (c *Controller) runnerOnline(ctx context.Context, t *Target, id int) string {
	if _, ok := ctx.Deadline(); !ok {
		return ""
	}
	out, err := c.cfg.Exec.Run(ctx, "glab", "api", fmt.Sprintf("runners/%d", id))
	if err != nil {
		return ""
	}
	var st struct {
		Online bool `json:"online"`
	}
	if json.Unmarshal(out, &st) == nil {
		if st.Online {
			return "yes"
		}
		return "no"
	}
	return ""
}

func (c *Controller) guardSupervisorOnlyTarget(t *Target, verb string) error {
	if !IsSupervisorOnlyTarget(t) {
		return nil
	}
	return errdefs.New("runners.gitlab."+verb, errdefs.CodeInvalidScope,
		"the gitlab-runner supervisor is not managed from a service-only row; use a registered runner for stop/remove, repair-service to fix the unit, or brew services for the formula service",
		nil)
}

func (c *Controller) warnServiceScope(ctx context.Context, t *Target) error {
	if t.EntryCount <= 1 {
		return nil
	}
	if c.cfg.AllowYes {
		return nil
	}
	pr := c.cfg.Prompter
	if pr == nil {
		if c.cfg.NonInteractive {
			return nil
		}
		pr = prompt.ForTTY()
	}
	msg := fmt.Sprintf("This service runs %d [[runners]] in %q. Continue?", t.EntryCount, t.ConfigPath)
	ok, err := pr.Confirm(ctx, msg)
	if err != nil {
		return err
	}
	if !ok {
		return errdefs.New("runners.gitlab", errdefs.CodeInvalidScope, "cancelled", nil)
	}
	return nil
}

func backupConfig(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	backup := path + ".runners-bak-" + time.Now().UTC().Format("20060102T150405Z")
	if err := os.WriteFile(backup, data, 0o600); err != nil {
		return "", err
	}
	return backup, nil
}

func unregisterBenign(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not registered")
}

func removeConfirmMessage(t *Target) string {
	if IsHelperServiceTarget(t) || (t != nil && t.Kind == TargetKindServiceOnly && strings.TrimSpace(t.Name) == "") {
		return fmt.Sprintf("Remove service unit %q from this machine?", labelOrDash(t.ServiceName))
	}
	if t.EntryCount > 1 {
		return fmt.Sprintf("Remove runner %q? Other [[runners]] in %q stay registered.", labelOrDash(t.Name), t.ConfigPath)
	}
	return fmt.Sprintf("Remove runner %q from this machine?", labelOrDash(t.Name))
}

func labelOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return strings.TrimSpace(s)
}
