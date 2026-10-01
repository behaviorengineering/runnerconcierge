package wizard

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/detect"
	"github.com/behaviorengineering/runnerconcierge/pkg/gitlabrunner"
	"github.com/behaviorengineering/runnerconcierge/pkg/install"
	"github.com/behaviorengineering/runnerconcierge/pkg/preset"
	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

// Options controls wizard execution.
type Options struct {
	ConfigPath      string
	Preset          preset.Preset
	NonInteractive  bool
	Resume          bool
	Fresh           bool
	AllowInstall    bool
	RunnerToken     string
	PAT             string
	ProjectPath     string
	Executor        string
	TagList         []string
	RunUntagged     bool
	WindowsPassword string
	RequireDocker   bool
	Prompter        prompt.Prompter
}

// Runner orchestrates setup stages.
type Runner struct {
	exec   *gitexec.Runner
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
	if found && !r.opts.Fresh && !r.opts.Resume && !r.opts.NonInteractive {
		msg := fmt.Sprintf("Incomplete setup at stage %q (runner id %d). Resume?", cp.Stage, cp.RunnerID)
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
	_ = r.store.Save(cp)

	token := strings.TrimSpace(r.opts.RunnerToken)
	runnerID := cp.RunnerID
	if token == "" && runnerID == 0 {
		client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)
		req := r.createRequest(cp)
		pat := strings.TrimSpace(r.opts.PAT)
		runnerID, token, err = client.CreateRunner(ctx, req, pat)
		if err != nil {
			return err
		}
		cp.RunnerID = runnerID
		cp.Description = req.Description
		cp.Tags = req.TagList
		cp.Executor = r.executor()
		_ = r.store.Save(cp)
	}
	if token == "" && runnerID > 0 {
		return fmt.Errorf("wizard: runner %d exists but glrt token missing; re-create in GitLab UI or pass --token", runnerID)
	}

	regArgs, err := gitlabrunner.BuildRegisterArgv(gitlabrunner.RegisterArgs{
		URL:              r.gitlabURL(),
		Token:            token,
		Name:             r.runnerName(),
		Executor:         r.executor(),
		ConfigPath:       cfgPath,
		WorkingDirectory: workDir,
	})
	if err != nil {
		return err
	}
	client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)
	if err := client.Register(ctx, runnerBin, regArgs); err != nil {
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
	if err := svc.Start(ctx); err != nil {
		return err
	}
	cp.Completed = appendUnique(cp.Completed, "service")
	_ = r.store.Save(cp)

	if runnerID > 0 {
		if err := client.WaitOnline(ctx, runnerID, 2*time.Minute); err != nil {
			return err
		}
	}
	cp.Completed = appendUnique(cp.Completed, "verify")
	cp.Stage = "done"
	return r.store.Save(cp)
}

func (r *Runner) gitlabURL() string {
	if r.preset.GitLabURL != "" {
		return r.preset.GitLabURL
	}
	return r.cfg.GitLabURL
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

func (r *Runner) createRequest(cp *state.Checkpoint) gitlabrunner.CreateRunnerRequest {
	tags := r.opts.TagList
	if len(tags) == 0 {
		tags = r.preset.TagList
	}
	runUntagged := r.opts.RunUntagged || r.preset.RunUntagged
	desc := strings.TrimSpace(r.cfg.DescriptionPrefix + " " + hostname())
	projectPath := r.opts.ProjectPath
	if projectPath == "" {
		projectPath = r.preset.RepoPath
	}
	req := gitlabrunner.CreateRunnerRequest{
		RunnerType:  "project_type",
		Description: desc,
		TagList:     tags,
		RunUntagged: runUntagged,
	}
	if projectPath != "" {
		pctx, pcancel := ensureDeadline(context.Background(), 2*time.Minute)
		defer pcancel()
		client := gitlabrunner.NewClient(r.gitlabURL(), r.exec, nil)
		id, err := client.ResolveProjectID(pctx, projectPath)
		if err == nil {
			req.ProjectID = id
		}
	}
	return req
}

func (r *Runner) runnerName() string {
	h, _ := os.Hostname()
	return strings.TrimSpace(h)
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

func ensureDeadline(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
