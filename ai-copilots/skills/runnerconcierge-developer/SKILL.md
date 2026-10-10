---
name: runnerconcierge-developer
description: >-
  Extend runnerconcierge packages, cobra verbs, service identity, and docker
  cleanup schedule. Use when changing pkg/cleanup, pkg/gitlab/runners,
  pkg/service, CLI wiring, or e2e fixtures. Not for operating a live runner.
---

# runnerconcierge-developer

**Moral:** Keep prune logic in `pkg/cleanup` and gitlab-runner lifecycle in `pkg/service`. The schedule invokes this binary; it MUST NOT ship OS shell scripts.

## Layout

- `cmd/runnerconcierge`: olly init, CLI entry
- `internal/cli`, `internal/wizard`, `internal/config`
- `pkg/cleanup`: one-shot prune + macOS LaunchAgent / Windows `schtasks` install
- `pkg/gitlab/runners`: forge-scoped control plane
- `pkg/service`: gitlab-runner Install/Start/Stop/Uninstall + `ListOwnership` identity
- `pkg/detect`, `pkg/install`, `pkg/gitlabrunner`, `pkg/inventory`, `pkg/repair`, `pkg/state`, `pkg/redact`, `pkg/preset`, `pkg/errdefs`, `pkg/forge`, `pkg/prompt`
- `internal/e2elive` (`e2e_live` build tag), `internal/e2elive/fixture`

## CLI surface

**CONSTRAINT:** Bare invoke MUST print the agent guide and MUST NOT start setup or a daemon.

- MUST: `cleanup` stay finite (one prune pass); interval lives in LaunchAgent / schtasks, not a long-running subcommand
- MUST: list `cleanup`, `runners gitlab list`, and `runners gitlab setup` in `printAgentGuide` / `printHelp`
- MUST: `--yes` on `cleanup` be persistent so `cleanup install --yes` works
- MUST NOT: treat `cleanup` as `service.Manager.Install` (that path talks to `gitlab-runner`)
- Enforcement: `go tool task smoke`; `cleanup --help` and `cleanup install --help` list `--yes`
- Violation: STOP, fix cobra flags, re-run smoke

CORRECT:
```text
runnerconcierge cleanup
runnerconcierge cleanup install --yes --interval 10m
```

PROHIBITED:
```text
# cobra --yes only on parent cleanup; install rejects unknown flag
# Manager.Install used to write a LaunchAgent that runs a .sh
```

## Cleanup package

**CONSTRAINT:** `wizard.Run` MUST call `cleanup.EnsureSchedule` after the gitlab-runner service starts (macOS/Windows, `AllowYes` true) so every setup installs the helper regardless of executor.

- MUST: no-op `EnsureSchedule` on Linux; non-interactive setup fails if schedule install fails on supported OS
- MUST NOT: gate cleanup install on docker executor only
- Enforcement: `pkg/wizard/wizard.go` after service start; checkpoint may include `cleanup_schedule`
- Violation: STOP, wire EnsureSchedule before verify/online wait

**CONSTRAINT:** Docker leftover prune MUST live in `pkg/cleanup` with `Config.Create()`.

- MUST: panic when `Exec` is nil; error when `Out` is nil
- MUST: call Docker through `gitexec.Exec`, not `os/exec` and not the Docker SDK
- MUST: skip with nil error when `docker info` fails (scheduled jobs MUST NOT flap)
- MUST: fail with `errdefs.CodeDockerUnavailable` when `ps` / `volume ls` fail after info succeeded
- MUST: portable unit name `runnerconcierge-docker-cleanup` (MUST NOT hardcode a host login or `com.<user>.*`)
- MUST: Darwin install replace LaunchAgents whose command basename is `gitlab-runner-docker-cleanup` only with `AllowYes`
- MUST: Windows install use `schtasks`, not a Windows Service
- MUST: Linux install/uninstall return `CodeUnsupportedOS`; `Run` remains allowed
- Enforcement: fake-exec tests in `pkg/cleanup`; `go test ./pkg/cleanup/...`
- Violation: STOP, restore Create/errdefs/skip rules, re-test

## Service identity

**CONSTRAINT:** `ListOwnership` MUST classify leftover GitLab-related units at discovery.

- MUST: `IsRunnerServiceName` include `runnerconcierge-docker-cleanup` and `gitlab-runner` / `runnerconcierge-e2e-`
- MUST: `ClassifyRole` treat cleanup unit (name or `runnerconcierge cleanup` argv) as `helper`
- MUST: match reason `runnerconcierge_cleanup` for the product helper
- MUST: Windows `ListOwnership` probe the scheduled task (kind `scheduled_task`); `ProcessUp` false between runs is valid
- MUST NOT: attach helper identity by guessing at Inspect time only
- Enforcement: `pkg/service` identity tests + `pkg/gitlab/runners` join/inspect tests
- Violation: STOP, copy identity in `JoinTargets` from Ownership

## gitlab-runner argv

**CONSTRAINT:** MUST load [gitlab-runner-cli/SKILL.md](../gitlab-runner-cli/SKILL.md) when changing `BuildRegisterArgv`, `BuildUnregisterArgv`, or adding flags executed via `gitlab-runner`.

- MUST: keep static argv tests plus `pkg/gitlabrunner/runner_compat_test.go` help contracts in sync with builders
- Enforcement: `go test ./pkg/gitlabrunner/...`
- Violation: STOP, follow gitlab-runner-cli skill, re-run tests

## Errors and tests

CLI: `runners gitlab list` is the forge-scoped control plane (`service.Manager.Stop`, `gitlabrunner.ListOwnedRunners` / `MatchRunner`); `runners gitlab setup` runs the wizard. Domain errors use `pkg/errdefs` (`Error()` omits cause argv; `FormatCLI` for stderr).

**Checkpoint vs secrets:** `pkg/state` checkpoint JSON MUST stay redacted (no glrt). Hydrate non-secret fields via `applyCheckpointToOpts`; identity stage saves `stage=identity` before doctor. glrt: `internal/config` keyring `GITLAB_RUNNER_TOKEN_<identity>` where identity is `parent-tag-hostname` from `RunnerIdentity`. Load identity or id slots only when checkpoint `runner_id` is set; tag/pending slots are resume fallbacks when identity cannot be computed. A new setup with `runner_id` 0 mints via API unless `--token`. Register `is not valid` resets or creates once, then retries. Tests use `NewMemKeyring()`.

**CONSTRAINT:** MUST keep quality gates green before claiming a Go change done.

- MUST: `gofmt`, `GOWORK=off go vet ./...`, `GOWORK=off go test -race -count=1 ./...`, `go tool task smoke`
- MUST: GitLab API tests use `httptest` in `pkg/gitlabrunner`
- MUST NOT: put host product paths or brand in this module
- Enforcement: `go tool task ci` locally when touching Go; smoke grep includes `cleanup --help`
- Violation: STOP, fix, re-run gates

## Live e2e env

| Variable | Purpose |
|----------|---------|
| `E2E_LIVE_MODE=fixture` | Run `TestFixtureLifecycle` |
| `E2E_LIVE_REQUIRE=1` | Skip → fatal on darwin/windows |
| `E2E_LIVE_ALLOW_EXISTING=1` | Allow seed when gitlab-runner services already exist |
| `E2E_LIVE_SET_WINDOWS_PASSWORD=1` | Allow `net user` password reset for repair (also on `GITHUB_ACTIONS`) |
| `RUNNERCONCIERGE_WINDOWS_PASSWORD` | Windows service install password for repair |
| `E2E_LIVE_ARTIFACT_DIR` | Write inventory JSON traces |

Workflow: `.github/workflows/e2e-live.yml` (`go tool task e2e-live` only).
