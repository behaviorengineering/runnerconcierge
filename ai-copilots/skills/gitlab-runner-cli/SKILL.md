---
name: gitlab-runner-cli
description: >-
  Build and validate argv for the external gitlab-runner binary (register,
  unregister, install, service verbs). Use when adding flags, debugging
  "flag provided but not defined", or extending pkg/gitlabrunner argv builders
  and runner_compat_test.go help contracts.
---

# gitlab-runner-cli

**Moral:** `gitlab-runner` is an external CLI with subcommand-specific flags. runnerconcierge MUST centralize argv in builders and prove flags against `subcommand --help` before users hit setup or remove at runtime.

## Ownership

| Concern | Location |
|---------|----------|
| Register argv | `pkg/gitlabrunner/register.go` → `BuildRegisterArgv` |
| Unregister argv | `pkg/gitlabrunner/unregister.go` → `BuildUnregisterArgv` |
| Exec register/unregister | `pkg/gitlabrunner/client.go` |
| Service install/start/stop | `pkg/service/service_darwin.go`, `service_windows.go` (inline argv today) |
| Help contract tests | `pkg/gitlabrunner/runner_compat_test.go` |

## Subcommand rules (19.x)

**CONSTRAINT:** `register` MUST pass `--non-interactive`. `unregister` MUST NOT pass `--non-interactive` (flag exists on register only; unregister rejects it).

**CONSTRAINT:** `--working-directory` MUST NOT appear on `register` argv (19.x rejects it). MUST pass working directory only on `gitlab-runner install` via `pkg/service` `InstallOpts`.

- Enforcement: `runner_compat_test.go`; `unregister_test.go` argv ban
- Violation: STOP, remove bad flag from builder, re-run `go test ./pkg/gitlabrunner/...`

CORRECT:
```text
register:   --non-interactive --url … --token … --name … --executor …
unregister: --name … [--config …] [--url …]
```

PROHIBITED:
```text
unregister --non-interactive --name …
```

**CONSTRAINT:** `BuildRegisterArgv` MUST NOT pass server-side runner fields on the CLI (`--tag-list`, `--run-untagged`). Tags and untagged policy MUST stay on GitLab API create (`CreateRunner`).

- Enforcement: `register_test.go`; register branch of `runner_compat_test.go`
- Violation: STOP, remove CLI flags, keep API form fields only

## Two-layer validation

**CONSTRAINT:** Every new or changed long flag on a builder MUST have both layers before claiming done.

1. **Static argv tests** (no binary): exact `[]string` or forbidden-flag checks in `*_test.go` beside the builder.
2. **Help contract tests** (binary on PATH): `gitlab-runner <subcommand> --help`, parse `--flag` names, assert each builder flag is listed; assert subcommand-specific policy (non-interactive, forbidden flags).

- Enforcement: `go test ./pkg/gitlabrunner/ -run 'Register|Unregister'`
- Violation: STOP, add missing layer, re-run tests

**CONSTRAINT:** Help contract tests MUST skip when `gitlab-runner` is not on PATH (`t.Skip`). MUST NOT fail CI on machines without the binary unless CI installs a pinned runner.

- Enforcement: `gitlabRunnerBin` in `runner_compat_test.go`
- Violation: STOP, use Skip not Fatal on LookPath failure

CORRECT:
```go
bin := gitlabRunnerBin(t) // skips if missing
helpFlags := subcommandFlagsFromHelp(t, bin, "register")
assertArgvFlagsListedInHelp(t, helpFlags, args, "register", "BuildRegisterArgv")
```

PROHIBITED:
```go
// Only happy-path manual test on one laptop; no static argv test
```

## Adding a new subcommand contract

**CONSTRAINT:** When runnerconcierge gains a new centralized argv builder for `gitlab-runner`, MUST add a `Test*Argv_matchesInstalledRunnerHelp` in `runner_compat_test.go` (or shared compat file) that:

- MUST: call `subcommandFlagsFromHelp(t, bin, "<subcommand>")`
- MUST: build a representative fixture argv from the builder
- MUST: assert every `--long` flag from the builder appears in help
- MAY: assert required core flags still exist on help (guard renames)
- MUST NOT: duplicate `gitlabRunnerBin` / help parsing; reuse helpers in `runner_compat_test.go`

For inline argv in `pkg/service` (install/start/stop), MUST either extract a `BuildInstallArgv` (or OS-specific builder) and test it, or add a `GOOS`-scoped compat test that mirrors the exact argv slice the platform manager uses.

## Debugging flag failures

| Symptom | Check |
|---------|--------|
| `flag provided but not defined: -non-interactive` on remove | Unregister argv; run `gitlab-runner unregister --help` |
| Register hangs or prompts | Missing `--non-interactive` on register argv |
| Tags wrong after setup | CLI `--tag-list` leaked; must be API-only |

**CONSTRAINT:** Before changing argv in response to a user error, MUST run `gitlab-runner <subcommand> --help` on the runner version they use and compare to the builder output.

- Enforcement: manual or compat test update in same change
- Violation: STOP, read help, then edit builder

## Quality gates

- MUST: `GOWORK=off go test -count=1 ./pkg/gitlabrunner/...` after argv changes
- MUST: `gofmt` on touched Go files
- MAY: note pinned runner version in PR when help text changes (e.g. 19.4.0)

## Cross-links

- Operator flows: [runnerconcierge-operator/SKILL.md](../runnerconcierge-operator/SKILL.md)
- Package layout and CI: [runnerconcierge-developer/SKILL.md](../runnerconcierge-developer/SKILL.md)
