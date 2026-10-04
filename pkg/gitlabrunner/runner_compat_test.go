package gitlabrunner

import (
	"context"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

var gitlabRunnerHelpFlagRE = regexp.MustCompile(`--([a-z][a-z0-9-]*)`)

func gitlabRunnerBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("gitlab-runner")
	if err != nil {
		t.Skip("gitlab-runner not on PATH")
	}
	return path
}

func subcommandFlagsFromHelp(t *testing.T, runnerBin, subcommand string) map[string]bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, runnerBin, subcommand, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("gitlab-runner %s --help: %v\n%s", subcommand, err, out)
	}
	text := string(out)
	flags := make(map[string]bool)
	for _, m := range gitlabRunnerHelpFlagRE.FindAllStringSubmatch(text, -1) {
		flags[m[1]] = true
	}
	if len(flags) == 0 {
		t.Fatalf("no flags parsed from %s --help:\n%s", subcommand, text)
	}
	return flags
}

func argvLongFlags(argv []string) []string {
	var names []string
	for _, a := range argv {
		if strings.HasPrefix(a, "--") {
			names = append(names, strings.TrimPrefix(a, "--"))
		}
	}
	return names
}

func assertArgvFlagsListedInHelp(t *testing.T, helpFlags map[string]bool, argv []string, subcommand, builder string) {
	t.Helper()
	for _, flag := range argvLongFlags(argv) {
		if !helpFlags[flag] {
			t.Fatalf("%s uses --%s but %s --help does not list it", builder, flag, subcommand)
		}
	}
}

// Contract tests against the installed gitlab-runner binary (skipped when not on PATH).

func TestUnregisterArgv_matchesInstalledRunnerHelp(t *testing.T) {
	bin := gitlabRunnerBin(t)
	helpFlags := subcommandFlagsFromHelp(t, bin, "unregister")

	if helpFlags["non-interactive"] {
		t.Fatal("unregister --help lists --non-interactive; BuildUnregisterArgv must not use it")
	}
	for _, required := range []string{"name", "config", "url"} {
		if !helpFlags[required] {
			t.Fatalf("unregister --help missing --%s (runner may have changed CLI)", required)
		}
	}

	args, err := BuildUnregisterArgv(UnregisterArgs{
		Name:       "contract-runner",
		ConfigPath: "/tmp/config.toml",
		URL:        "https://gitlab.example",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range argvLongFlags(args) {
		if !helpFlags[flag] {
			t.Fatalf("BuildUnregisterArgv uses --%s but unregister --help does not list it", flag)
		}
	}
}

func TestRegisterArgv_matchesInstalledRunnerHelp(t *testing.T) {
	bin := gitlabRunnerBin(t)
	helpFlags := subcommandFlagsFromHelp(t, bin, "register")

	if !helpFlags["non-interactive"] {
		t.Fatal("register --help missing --non-interactive")
	}
	for _, required := range []string{"url", "token", "name", "executor", "config"} {
		if !helpFlags[required] {
			t.Fatalf("register --help missing --%s (runner may have changed CLI)", required)
		}
	}

	cases := []struct {
		name   string
		in     RegisterArgs
		forbid []string
	}{
		{
			name: "shell",
			in: RegisterArgs{
				URL:              "https://gitlab.example",
				Token:            "glrt-contract",
				Name:             "contract-shell",
				Executor:         "shell",
				Shell:            "bash",
				ConfigPath:       "/tmp/config.toml",
				WorkingDirectory: "/tmp/work",
			},
			forbid: []string{"tag-list", "run-untagged", "working-directory"},
		},
		{
			name: "docker",
			in: RegisterArgs{
				URL:         "https://gitlab.example",
				Token:       "glrt-contract",
				Name:        "contract-docker",
				Executor:    "docker",
				ConfigPath:  "/tmp/config.toml",
				DockerImage: "alpine:latest",
			},
			forbid: []string{"tag-list", "run-untagged", "working-directory"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := BuildRegisterArgv(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			assertArgvFlagsListedInHelp(t, helpFlags, args, "register", "BuildRegisterArgv")
			flags := argvLongFlags(args)
			if !slices.Contains(flags, "non-interactive") {
				t.Fatal("register argv must include --non-interactive")
			}
			for _, f := range tc.forbid {
				if slices.Contains(flags, f) {
					t.Fatalf("server-side flags must not appear in register argv: --%s", f)
				}
			}
		})
	}
}
