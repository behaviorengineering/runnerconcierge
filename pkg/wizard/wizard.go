package wizard

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/cleanup"
	"github.com/behaviorengineering/runnerconcierge/pkg/detect"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/gitlabrunner"
	"github.com/behaviorengineering/runnerconcierge/pkg/install"
	"github.com/behaviorengineering/runnerconcierge/pkg/preset"
	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

// Options controls wizard execution.
type Options struct {
	ConfigPath        string
	Preset            preset.Preset
	NonInteractive    bool
	Resume            bool
	Fresh             bool
	AllowInstall      bool
	RunnerToken       string
	IgnoreKeyringGLRT bool // wizard sets after archiving stage=done; mint fresh glrt
	PAT               string
	Keyring           operatorconfig.Keyring // nil → OS keyring
	ProjectPath       string
	GroupPath         string
	RunnerType        string
	Executor          string
	TagList           []string
	RunUntagged       bool
	WindowsPassword   string
	RequireDocker     bool
	Prompter          prompt.Prompter
}

// Runner orchestrates setup stages.
type Runner struct {
	exec   gitexec.Exec
	cfg    *config.UserConfig
	store  *state.Store
	preset preset.Preset
	opts   Options
	out    func(string)
}

// New builds a wizard runner.
func New(opts Options, out func(string)) (*Runner, error) {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return nil, err
	}
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = func(string) {}
	}
	return &Runner{
		exec:   gitexec.New(),
		cfg:    cfg,
		store:  &state.Store{Dir: dir},
		preset: opts.Preset,
		opts:   opts,
		out:    out,
	}, nil
}

// Run executes the setup pipeline.
func (r *Runner) Run(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("wizard: runner is nil")
	}
	ctx, cancel := ensureDeadline(ctx, 30*time.Minute)
	defer cancel()

	pr := r.opts.Prompter
	if pr == nil {
		if r.opts.NonInteractive {
			pr = &prompt.NonInteractive{}
		} else {
			pr = prompt.ForTTY()
		}
	}

	if r.opts.Fresh {
		_ = r.store.Archive()
	}

	cp, found, err := r.store.Load()
	if err != nil {
		return err
	}
	cp, found = r.beginFromStoredCheckpoint(cp, found)
	if found && !r.opts.Fresh && !r.opts.Resume && !r.opts.NonInteractive {
		tagHint := ""
		if cp != nil && len(cp.Tags) > 0 {
			tagHint = fmt.Sprintf(", tag %q", strings.TrimSpace(cp.Tags[0]))
		}
		msg := fmt.Sprintf("Incomplete setup at stage %q (runner id %d%s). Resume?", cp.Stage, cp.RunnerID, tagHint)
		ok, err := pr.Confirm(ctx, msg)
		if err != nil {
			return err
		}
		if !ok {
			_ = r.store.Archive()
			found = false
		} else {
			r.opts.Resume = true
		}
	}
	if !r.opts.Resume || !found {
		cp = &state.Checkpoint{
			Version:     1,
			Fingerprint: detect.Fingerprint(),
			GitLabURL:   r.gitlabURL(),
		}
	}

	if found && !r.opts.Fresh {
		applyCheckpointToOpts(&r.opts, cp)
		r.outResumeSummary(cp)
	}

	if err := r.promptRunnerName(ctx, pr, cp); err != nil {
		return err
	}
	if len(r.opts.TagList) > 0 {
		cp.Tags = append([]string(nil), r.opts.TagList...)
	}
	cp.Stage = "identity"
	if err := r.saveCheckpoint(cp); err != nil {
		return err
	}

	if runtime.GOOS == "windows" && r.opts.WindowsPassword == "" && !r.opts.NonInteractive {
		pw, err := pr.Password(ctx, "Windows service account password (login user, not shown again)")
		if err != nil {
			return err
		}
		r.opts.WindowsPassword = pw
	}

	doc := detect.NewDoctor(r.exec)
	dctx, dcancel := detect.WithDoctorDeadline(ctx)
	rep, err := doc.Run(dctx, r.opts.RequireDocker)
	dcancel()
	if err != nil {
		return err
	}
	for _, iss := range rep.Issues {
		if iss.Block {
			return fmt.Errorf("doctor: %s", iss.Message)
		}
		r.out("doctor: " + iss.Message)
	}
	cp.Completed = appendUnique(cp.Completed, "doctor")

	inst := install.New(r.exec)
	if _, err := inst.Ensure(ctx, install.ToolGlab, r.opts.AllowInstall || r.opts.NonInteractive); err != nil {
		return err
	}
	runnerBin, err := inst.Ensure(ctx, install.ToolGitLabRunner, r.opts.AllowInstall || r.opts.NonInteractive)
	if err != nil {
		return err
	}
	_ = install.LongPathsWindows(ctx, r.exec)
	cp.BinaryPath = runnerBin
	cp.Completed = appendUnique(cp.Completed, "tools")

	cfgPath, workDir := r.paths()
	cp.ConfigPath = cfgPath
	if err := r.saveCheckpoint(cp); err != nil {
		return err
	}

	var glUser gitlabrunner.GitLabUser
	if !r.hasRunnerToken(cp) {
		glUser, err = r.ensureLoggedIn(ctx, pr)
		if err != nil {
			return err
		}
	}

	if r.needsScopeHydration(cp) {
		if err := r.hydrateRunnerScopeFromGitLab(ctx, cp); err != nil {
			if !r.hasRunnerToken(cp) {
				return err
			}
			r.out("setup: could not load runner scope from GitLab: " + err.Error())
		}
	}

	if err := r.promptSetupOptions(ctx, pr, cp); err != nil {
		return err
	}
	if err := r.hydrateRunnerToken(cp); err != nil {
		return err
	}
	if !r.hasRunnerToken(cp) {
		if err := r.ensureRunnerAccess(ctx, glUser, cp); err != nil {
			return err
		}
		if err := r.resetRunnerGLRTIfNeeded(ctx, cp); err != nil {
			return err
		}
	}

	token, _, err := r.resolveRegisterToken(ctx, cp)
	if err != nil {
		return err
	}
	if err := r.registerWithRetry(ctx, pr, runnerBin, cp, token, cfgPath, workDir); err != nil {
		return err
	}
	cp.Completed = appendUnique(cp.Completed, "register")
	_ = r.store.Save(cp)

	svc := service.New(r.exec)
	_ = service.EnsureSingleProcess(ctx, r.exec)
	if err := svc.Install(ctx, service.InstallOpts{
		BinaryPath:       runnerBin,
		ConfigPath:       cfgPath,
		WorkingDirectory: workDir,
		WindowsUser:      os.Getenv("USERNAME"),
		WindowsPassword:  r.opts.WindowsPassword,
		UseBrewServices:  runtime.GOOS == "darwin",
	}); err != nil && !r.opts.NonInteractive {
		r.out("service install: " + err.Error())
	} else if err != nil {
		return err
	}
	if err := svc.Start(ctx, service.StartOpts{}); err != nil {
		return err
	}
	cp.Completed = appendUnique(cp.Completed, "service")
	_ = r.store.Save(cp)

	if err := cleanup.EnsureSchedule(ctx, r.exec, lineOut(r.out), true); err != nil {
		if r.opts.NonInteractive {
			return errdefs.New("setup", errdefs.CodeServiceStart, "could not install docker cleanup schedule", err)
		}
		r.out("cleanup schedule: " + err.Error())
	} else if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		cp.Completed = appendUnique(cp.Completed, "cleanup_schedule")
		_ = r.store.Save(cp)
	}

	if err := r.verifyRunnerOnlineOnGitLab(ctx, cp.RunnerID); err != nil {
		return err
	}
	cp.Completed = appendUnique(cp.Completed, "verify")
	cp.Stage = state.CheckpointStageDone
	return r.store.Save(cp)
}

// beginFromStoredCheckpoint archives a finished checkpoint so setup can register a new runner.
func (r *Runner) beginFromStoredCheckpoint(cp *state.Checkpoint, found bool) (*state.Checkpoint, bool) {
	if r == nil || !found || r.opts.Fresh || cp == nil {
		return cp, found
	}
	if cp.Stage != state.CheckpointStageDone {
		return cp, found
	}
	parts := []string{"setup: previous setup is complete"}
	if cp.RunnerID > 0 {
		parts = append(parts, fmt.Sprintf("runner_id=%d", cp.RunnerID))
	}
	if tag := canonicalTag(r.opts, cp); tag != "" {
		parts = append(parts, fmt.Sprintf("tag=%q", tag))
	} else if len(cp.Tags) > 0 {
		if t := strings.TrimSpace(cp.Tags[0]); t != "" {
			parts = append(parts, fmt.Sprintf("tag=%q", t))
		}
	}
	r.out(strings.Join(parts, " ") + "; starting a new runner setup")
	_ = r.store.Archive()
	r.opts.IgnoreKeyringGLRT = true
	return cp, false
}

func (r *Runner) outResumeSummary(cp *state.Checkpoint) {
	if r == nil || cp == nil {
		return
	}
	parts := []string{fmt.Sprintf("resume stage=%q", cp.Stage)}
	if cp.RunnerID > 0 {
		parts = append(parts, fmt.Sprintf("runner_id=%d", cp.RunnerID))
	}
	if tag := canonicalTag(r.opts, cp); tag != "" {
		parts = append(parts, fmt.Sprintf("tag=%q", tag))
	}
	if e := strings.TrimSpace(cp.Executor); e != "" {
		parts = append(parts, fmt.Sprintf("executor=%q", e))
	}
	if p := strings.TrimSpace(cp.RepoPath); p != "" {
		parts = append(parts, fmt.Sprintf("project=%q", p))
	}
	if g := strings.TrimSpace(cp.GroupPath); g != "" {
		parts = append(parts, fmt.Sprintf("group=%q", g))
	}
	r.out("setup: " + strings.Join(parts, " "))
}

func (r *Runner) gitlabURL() string {
	if r.preset.GitLabURL != "" {
		return r.preset.GitLabURL
	}
	return r.cfg.GitLabURL
}

func (r *Runner) dockerImage() string {
	if strings.TrimSpace(r.preset.DockerImage) != "" {
		return strings.TrimSpace(r.preset.DockerImage)
	}
	if r.executor() != "docker" {
		return ""
	}
	return "docker:24"
}

func (r *Runner) executor() string {
	if r.preset.Executor != "" {
		return r.preset.Executor
	}
	if r.opts.Executor != "" {
		return r.opts.Executor
	}
	return r.cfg.DefaultExecutor
}

func (r *Runner) createRequest(ctx context.Context, cp *state.Checkpoint) (gitlabrunner.CreateRunnerRequest, error) {
	tags := r.opts.TagList
	if len(tags) == 0 {
		tags = r.preset.TagList
	}
	runUntagged := r.opts.RunUntagged || r.preset.RunUntagged
	desc := r.runnerDescription(cp)
	client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)

	req := gitlabrunner.CreateRunnerRequest{
		Description: desc,
		TagList:     tags,
		RunUntagged: runUntagged,
	}

	switch r.effectiveRunnerType() {
	case "group_type":
		req.RunnerType = "group_type"
		if r.preset.GroupID > 0 {
			req.GroupID = r.preset.GroupID
			return req, nil
		}
		groupPath := strings.TrimSpace(r.opts.GroupPath)
		if groupPath == "" {
			return req, errdefs.New("setup", errdefs.CodeInvalidScope, "GitLab group path is required (--group)", nil)
		}
		id, err := client.ResolveGroupID(ctx, groupPath)
		if err != nil {
			return req, errdefs.New("setup", errdefs.CodeOf(err), "could not resolve GitLab group id", err)
		}
		req.GroupID = id
		return req, nil
	default:
		req.RunnerType = "project_type"
		projectPath := strings.TrimSpace(r.opts.ProjectPath)
		if projectPath == "" {
			projectPath = strings.TrimSpace(r.preset.RepoPath)
		}
		if projectPath == "" {
			return req, errdefs.New("setup", errdefs.CodeInvalidScope, "GitLab project path is required (--repo)", nil)
		}
		id, err := client.ResolveProjectID(ctx, projectPath)
		if err != nil {
			return req, errdefs.New("setup", errdefs.CodeOf(err), "could not resolve GitLab project id", err)
		}
		req.ProjectID = id
		return req, nil
	}
}

func (r *Runner) effectiveRunnerType() string {
	if t := strings.TrimSpace(r.opts.RunnerType); t != "" {
		return t
	}
	if t := strings.TrimSpace(r.preset.RunnerType); t != "" {
		return t
	}
	if strings.TrimSpace(r.opts.GroupPath) != "" || r.preset.GroupID > 0 {
		return "group_type"
	}
	if strings.TrimSpace(r.opts.ProjectPath) != "" || strings.TrimSpace(r.preset.RepoPath) != "" {
		return "project_type"
	}
	return ""
}

func (r *Runner) runnerName(cp *state.Checkpoint) string {
	if id, err := runnerIdentityFor(r.opts, cp); err == nil {
		return id
	}
	h, _ := os.Hostname()
	return strings.TrimSpace(h)
}

func (r *Runner) runnerDescription(cp *state.Checkpoint) string {
	name := r.runnerName(cp)
	if p := strings.TrimSpace(r.cfg.DescriptionPrefix); p != "" {
		return strings.TrimSpace(p + " " + name)
	}
	return name
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

func (r *Runner) paths() (configPath, workDir string) {
	cfg, work, _ := service.DefaultPaths()
	if cfg == "" {
		home, _ := os.UserHomeDir()
		cfg = home + "/.gitlab-runner/config.toml"
		work = home
	}
	return cfg, work
}

func appendUnique(list []string, item string) []string {
	for _, v := range list {
		if v == item {
			return list
		}
	}
	return append(list, item)
}

type lineOutWriter struct {
	fn func(string)
}

func (w lineOutWriter) Write(p []byte) (int, error) {
	if w.fn != nil && len(p) > 0 {
		w.fn(strings.TrimRight(string(p), "\n"))
	}
	return len(p), nil
}

func lineOut(fn func(string)) io.Writer {
	if fn == nil {
		return io.Discard
	}
	return lineOutWriter{fn: fn}
}

func ensureDeadline(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
