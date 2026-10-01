package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/wizard"
	"github.com/behaviorengineering/runnerconcierge/pkg/detect"
	"github.com/behaviorengineering/runnerconcierge/pkg/preset"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

const (
	ExitOK       = 0
	ExitUsage    = 2
	ExitFail     = 1
	ExitDoctor   = 3
)

var version = "dev"

// SetVersion injects release identity from main.
func SetVersion(v string) {
	version = strings.TrimSpace(v)
	if version == "" {
		version = "dev"
	}
}

// Run dispatches argv.
func Run(ctx context.Context, args []string, w io.Writer) int {
	if len(args) == 0 {
		return runWizard(ctx, w, preset.Preset{}, nil, false)
	}
	switch args[0] {
	case "help", "-h", "--help":
		printHelp(w)
		return ExitOK
	case "version":
		fmt.Fprintf(w, "runnerconcierge %s\n", version)
		return ExitOK
	case "init":
		return runInit(args[1:], w)
	case "doctor":
		return runDoctor(ctx, args[1:], w)
	case "verify":
		return runVerify(ctx, args[1:], w)
	case "setup":
		return runSetup(ctx, args[1:], w, preset.Preset{})
	default:
		printHelp(w)
		return ExitUsage
	}
}

func printAgentGuide(w io.Writer) {
	writef(w, "runnerconcierge %s: GitLab self-hosted runner setup wizard\n", version)
	writef(w, "")
	writef(w, "Agent docs: AGENTS.md, ai-copilots/README.md, ai-copilots/BOOTSTRAP.md")
	writef(w, "Operator skill: ai-copilots/skills/runnerconcierge-operator/SKILL.md")
	writef(w, "")
	writef(w, "Bare invoke runs the interactive setup wizard.")
	writef(w, "Commands: init, doctor, verify, setup, version, help")
	writef(w, "Automation: runnerconcierge setup --non-interactive --yes ...")
}

func printHelp(w io.Writer) {
	writef(w, "Usage: runnerconcierge [command] [flags]")
	writef(w, "Commands:")
	writef(w, "  (bare)   interactive setup wizard")
	writef(w, "  init     seed user config.yaml")
	writef(w, "  doctor   preflight report")
	writef(w, "  verify   service status")
	writef(w, "  setup    wizard with flags")
	writef(w, "  version  release identity")
	writef(w, "  help     this catalog")
}

func runInit(args []string, w io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	force := fs.Bool("force", false, "overwrite config.yaml")
	_ = fs.Parse(args)
	path, err := config.Init(*force)
	if err != nil {
		writef(w, "init: %v\n", err)
		return ExitFail
	}
	writef(w, "config: %s\n", path)
	return ExitOK
}

func runDoctor(ctx context.Context, args []string, w io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	docker := fs.Bool("docker", false, "require docker")
	_ = fs.Parse(args)
	ctx, cancel := withDeadline(ctx, 30*time.Second)
	defer cancel()
	doc := detect.NewDoctor(gitexec.New())
	rep, err := doc.Run(ctx, *docker)
	if err != nil {
		writef(w, "doctor: %v\n", err)
		return ExitFail
	}
	writef(w, "os=%s arch=%s user=%s elevated=%v git=%v runner=%s glab=%s\n",
		rep.GOOS, rep.GOARCH, rep.Username, rep.Elevated, rep.GitOK, rep.RunnerVer, rep.GlabVer)
	blocked := false
	for _, iss := range rep.Issues {
		writef(w, "- [%s] %s\n", iss.Code, iss.Message)
		if iss.Block {
			blocked = true
		}
	}
	if blocked {
		return ExitDoctor
	}
	return ExitOK
}

func runVerify(ctx context.Context, args []string, w io.Writer) int {
	_ = flag.NewFlagSet("verify", flag.ContinueOnError)
	ctx, cancel := withDeadline(ctx, 30*time.Second)
	defer cancel()
	svc := service.New(gitexec.New())
	st, err := svc.Status(ctx)
	if err != nil {
		writef(w, "verify: %v\n", err)
		return ExitFail
	}
	writef(w, "%s\n", st)
	return ExitOK
}

func runSetup(ctx context.Context, args []string, w io.Writer, pre preset.Preset) int {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	nonInteractive := fs.Bool("non-interactive", false, "no prompts")
	resume := fs.Bool("resume", false, "resume checkpoint")
	fresh := fs.Bool("fresh", false, "discard checkpoint")
	yes := fs.Bool("yes", false, "allow installs")
	token := fs.String("token", "", "glrt runner token")
	pat := fs.String("pat", "", "gitlab PAT with create_runner")
	project := fs.String("repo", "", "group/project path")
	executor := fs.String("executor", "", "shell or docker")
	tags := fs.String("tag-list", "", "comma-separated tags")
	configPath := fs.String("config", "", "config.yaml path")
	_ = fs.Parse(args)
	tagList := splitTags(*tags)
	if len(tagList) == 0 {
		tagList = pre.TagList
	}
	return runWizard(ctx, w, pre, &wizard.Options{
		ConfigPath:     *configPath,
		Preset:         pre,
		NonInteractive: *nonInteractive,
		Resume:         *resume,
		Fresh:          *fresh,
		AllowInstall:   *yes,
		RunnerToken:    *token,
		PAT:            *pat,
		ProjectPath:    *project,
		Executor:       *executor,
		TagList:        tagList,
	}, false)
}

func runWizard(ctx context.Context, w io.Writer, pre preset.Preset, opt *wizard.Options, showGuide bool) int {
	if showGuide {
		printAgentGuide(w)
	}
	if opt == nil {
		opt = &wizard.Options{Preset: pre}
	}
	opt.Preset = pre
	run, err := wizard.New(*opt, func(s string) { writef(w, "%s\n", s) })
	if err != nil {
		writef(w, "setup: %v\n", err)
		return ExitFail
	}
	ctx, cancel := withDeadline(ctx, 45*time.Minute)
	defer cancel()
	if err := run.Run(ctx); err != nil {
		writef(w, "setup: %v\n", err)
		return ExitFail
	}
	writef(w, "setup complete\n")
	return ExitOK
}

func withDeadline(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

func splitTags(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func writef(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, format, args...)
}

// PresetFromEnv loads optional preset path (host binaries may inject via build).
func PresetFromEnv() preset.Preset {
	return preset.Preset{}
}

// Main is used by cmd for bare invoke.
func Main(ctx context.Context) int {
	return Run(ctx, os.Args[1:], os.Stdout)
}
