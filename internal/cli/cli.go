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
	"github.com/behaviorengineering/runnerconcierge/pkg/detect"
	"github.com/behaviorengineering/runnerconcierge/pkg/preset"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
	"github.com/behaviorengineering/runnerconcierge/pkg/wizard"
)

const (
	ExitOK     = 0
	ExitUsage  = 2
	ExitFail   = 1
	ExitDoctor = 3
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
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	if len(args) == 0 {
		printAgentGuide(stdout)
		return ExitOK
	}
	if code, ok := dispatchCobra(ctx, args, stdout, stderr); ok {
		return code
	}
	printHelp(stderr)
	return ExitUsage
}

func printAgentGuide(w io.Writer) {
	writef(w, "runnerconcierge %s: GitLab self-hosted runner setup CLI\n", version)
	writef(w, "")
	writef(w, "Purpose: install, register, and run gitlab-runner under the login user on macOS and Windows.")
	writef(w, "")
	writef(w, "Agent docs: AGENTS.md, ai-copilots/README.md, ai-copilots/BOOTSTRAP.md")
	writef(w, "Operator skill: ai-copilots/skills/runnerconcierge-operator/SKILL.md")
	writef(w, "")
	writef(w, "Inspect (read-only):")
	writef(w, "  doctor, verify, status, version, help")
	writef(w, "Execute (mutates host or GitLab):")
	writef(w, "  init, setup, repair-service")
	writef(w, "")
	writef(w, "Automation: runnerconcierge setup --non-interactive --yes --repo group/project ...")
	writef(w, "Flags: --yes allows package installs; pass --repo for project runners.")
}

func printHelp(w io.Writer) {
	writef(w, "Usage: runnerconcierge <command> [flags]")
	writef(w, "Commands:")
	writef(w, "  init     seed user config.yaml")
	writef(w, "  doctor   preflight report")
	writef(w, "  verify   service status")
	writef(w, "  status   runner inventory and smells")
	writef(w, "  repair-service  rebind service to login user")
	writef(w, "  setup    setup wizard with flags")
	writef(w, "  version  release identity")
	writef(w, "  help     this catalog")
	writef(w, "")
	writef(w, "Bare invoke prints the agent operating guide (no setup).")
}

func runDoctor(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	docker := fs.Bool("docker", false, "require docker")
	_ = fs.Parse(args)
	ctx, cancel := withDeadline(ctx, 30*time.Second)
	defer cancel()
	doc := detect.NewDoctor(gitexec.New())
	rep, err := doc.Run(ctx, *docker)
	if err != nil {
		writef(stderr, "doctor: %v\n", err)
		return ExitFail
	}
	writef(stdout, "os=%s arch=%s user=%s elevated=%v git=%v runner=%s glab=%s\n",
		rep.GOOS, rep.GOARCH, rep.Username, rep.Elevated, rep.GitOK, rep.RunnerVer, rep.GlabVer)
	blocked := false
	for _, iss := range rep.Issues {
		writef(stdout, "- [%s] %s\n", iss.Code, iss.Message)
		if iss.Block {
			blocked = true
		}
	}
	if blocked {
		return ExitDoctor
	}
	return ExitOK
}

func runVerify(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	_ = flag.NewFlagSet("verify", flag.ContinueOnError)
	ctx, cancel := withDeadline(ctx, 30*time.Second)
	defer cancel()
	svc := service.New(gitexec.New())
	st, err := svc.Status(ctx)
	if err != nil {
		writef(stderr, "verify: %v\n", err)
		return ExitFail
	}
	writef(stdout, "%s\n", st)
	return ExitOK
}

func runWizard(ctx context.Context, stdout, stderr io.Writer, pre preset.Preset, opt *wizard.Options) error {
	if opt == nil {
		opt = &wizard.Options{Preset: pre}
	}
	opt.Preset = pre
	run, err := wizard.New(*opt, func(s string) { writef(stdout, "%s\n", s) })
	if err != nil {
		return err
	}
	ctx, cancel := withDeadline(ctx, 45*time.Minute)
	defer cancel()
	if err := run.Run(ctx); err != nil {
		return err
	}
	writef(stdout, "setup complete\n")
	return nil
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
	return Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
}
