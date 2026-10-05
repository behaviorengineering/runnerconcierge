---
name: runnerconcierge-operator
description: >-
  Operate runnerconcierge: doctor, status, repair-service, cleanup, and
  runners gitlab list/setup on macOS and Windows. Use for local GitLab runner inventory,
  docker leftover prune, and day-2 stop/start/repair/remove. Not for GitHub Actions.
---

# runnerconcierge-operator

**Moral:** Inspect the machine, then act with an explicit verb. Bare `runnerconcierge` prints the agent guide and MUST NOT run setup.

## Start every task

1. Confirm this checkout is `github.com/behaviorengineering/runnerconcierge` (`git rev-parse --show-toplevel`).
2. Build when needed: `make build`.
3. Prefer inspect commands before anything that stops, removes, or rebinds a unit.

## Inspect then act

**CONSTRAINT:** MUST inspect before a destructive or service-changing command.

- MUST: run `status` (full dump) or `runners gitlab list --json` / interactive Inspect before `repair-service`, `runners gitlab list --action stop|start|repair|remove`, `cleanup uninstall`, or `cleanup install --yes`
- MUST NOT: skip inspect because a picker label looks like a registered runner
- Enforcement: the last inspect output is in the thread (or `--json` was run) before the act
- Violation: STOP, run inspect, then continue

CORRECT:
```text
runnerconcierge status
runnerconcierge runners gitlab list --json
runnerconcierge cleanup
runnerconcierge cleanup install --yes
```

PROHIBITED:
```text
runnerconcierge runners gitlab list --action remove --yes
# no inspect; brew gitlab-runner treated as a fourth GitLab registration
```

## Units on this machine

`runners gitlab list` lists **registered** `[[runners]]` from `config.toml` plus leftover **service-only** units.

| Kind | Typical row | What it does |
|------|-------------|--------------|
| Registered runner | name from TOML, linked service or not | GitLab job identity |
| Supervisor | brew/windows `gitlab-runner` | Long-lived `gitlab-runner run`; one process serves every `[[runners]]` in that config. **Not** a separate `runners gitlab` picker row; it appears on registered runner rows (start/stop/remove last runner). Use `repair-service`, `verify`, or `brew services` for the unit alone. |
| Helper | `runnerconcierge-docker-cleanup` (or a legacy script LaunchAgent) | Periodic docker leftover prune; not a GitLab registration |
| Fixture | `runnerconcierge-e2e-*` | Live e2e only |

**CONSTRAINT:** MUST treat a brew or Windows `gitlab-runner` supervisor as the CI engine, not as an extra GitLab runner name.

- MUST NOT: stop or remove the supervisor to "clean up unused runners" when registrations still need jobs
- MUST: use Inspect Role / Why listed / Command / Process up for service-only rows
- Enforcement: Role is `supervisor` or `helper` before stop/remove
- Violation: STOP, Inspect, then ask the operator

`Process up` on an interval helper is often **no** between ticks. That is expected.

## Commands

Read-only (safe to run first):

- `runnerconcierge`: agent operating guide (no setup)
- `runnerconcierge doctor`: setup preflight (`--docker` when docker executor is required)
- `runnerconcierge verify`: one-line service status
- `runnerconcierge status`: inventory and smells (`--json` for agents)
- `runnerconcierge version`, `help`

Changes this machine or GitLab:

- `runnerconcierge init`: seed user `config.yaml`
- `runnerconcierge runners gitlab setup`: wizard (`--repo group/project` or `--group my-group` for automation; interactive TTY asks project vs group scope, then lists membership). On macOS and Windows, setup also installs the `runnerconcierge-docker-cleanup` schedule (all executor types; safe when Docker is unused)
- `runnerconcierge repair-service --runner-config PATH`: rebind the **gitlab-runner** service to the login user (`--yes`; Windows password via prompt, `--windows-password`, or `RUNNERCONCIERGE_WINDOWS_PASSWORD`)
- `runnerconcierge cleanup`: one-shot prune of exited `runner-*` containers older than `--min-age` (default 1h) and dangling `runner-*` volumes. Docker down → skip, exit 0
- `runnerconcierge cleanup install --yes`: register the periodic helper (macOS LaunchAgent, Windows Task Scheduler). `--yes` replaces legacy helpers whose command basename is `gitlab-runner-docker-cleanup`
- `runnerconcierge cleanup uninstall`: remove the product helper `runnerconcierge-docker-cleanup` only
- `runnerconcierge runners gitlab list`: control plane (`--json`, `--name`, `--service`, `--config`, `--action inspect|stop|start|repair|remove`, `--yes`, `--local`, `--id`)

**CONSTRAINT:** MUST pass a forge and subcommand on `runners`.

- MUST: `runners gitlab list` or `runners gitlab setup` (or `runners github`, which fails closed as unimplemented)
- MUST NOT: invoke `runnerconcierge runners` or bare `runners gitlab` with no `list`/`setup`
- Enforcement: missing forge exits non-zero with `choose a forge`; bare `runners gitlab` exits with `choose a subcommand`
- Violation: STOP, add `gitlab list` or `gitlab setup`

## Cleanup (macOS and Windows)

Same prune command on both OSes. Install is the OS schedule, not a shell script.

| OS | Install mechanism | Unit name |
|----|-------------------|-----------|
| macOS | LaunchAgent, `StartInterval` from `--interval` (default 10m) | `runnerconcierge-docker-cleanup` |
| Windows | `schtasks` minute trigger, login user, `/RL LIMITED` | `runnerconcierge-docker-cleanup` |
| Linux | `cleanup` run is supported | install/uninstall return `unsupported_os` |

**CONSTRAINT:** MUST install the Go helper; MUST NOT add bash, zsh, or PowerShell prune scripts.

- MUST: rely on `runners gitlab setup` to install the schedule on macOS/Windows; use `cleanup install --yes` for existing installs or to repair the helper
- MUST NOT: write `~/bin/gitlab-runner-docker-cleanup` or a host-branded LaunchAgent label
- Enforcement: helper command argv is this binary plus `cleanup`
- Violation: STOP, uninstall the script helper with `--yes`, install the product unit

CORRECT:
```text
runnerconcierge cleanup
runnerconcierge cleanup install --yes
runnerconcierge cleanup uninstall
```

PROHIBITED:
```text
# new zsh/ps1 sidecar that docker prune's runner leftovers
```

## Setup and Windows logon

**CONSTRAINT:** MUST install the gitlab-runner **service** as the interactive login user.

- MUST: never LocalSystem
- MUST: run `doctor` before `runners gitlab setup` on a new host
- MUST: pass tags only via GitLab `POST /user/runners`, not `gitlab-runner register`
- MUST NOT: commit PAT or `glrt` tokens
- Enforcement: `status` findings for LocalSystem / wrong user; secrets stay in keyring or env
- Violation: STOP, `repair-service`, do not leave LocalSystem running jobs

### Runner token (glrt) and checkpoint resume

Interactive `runners gitlab setup` runs an **identity** step **before** doctor and tool install: hydrate empty flags from `state.json` (tags, executor, `repo_path`, `group_path`), then **runner tag**, then **glrt** (hidden). Leave glrt empty to create a new runner via GitLab API. Store glrt in the OS keyring (service `runnerconcierge`) **before** register. If checkpoint has `runner_id` but keyring miss, setup **reprompts** for glrt before register (TTY only).

| Account / env | When |
|---------------|------|
| `GITLAB_RUNNER_TOKEN_<tag>` | Primary: glrt for canonical tag (first `--tag-list` or interactive tag prompt) |
| `GITLAB_RUNNER_TOKEN_<id>` | Also written after `POST /user/runners` for resume by runner id |
| `GITLAB_RUNNER_TOKEN` | Legacy pending slot (read-only fallback) |
| `GITLAB_TOKEN` | PAT for create_runner (optional; glab auth may suffice) |

- MUST: skip the glrt prompt when `--token`, env, or keyring already has the token for that tag (save-and-forget)
- MUST: resume failed setup using checkpoint `runner_id` plus keyring glrt; `state.json` stays redacted (no tokens)
- MUST NOT: echo or log glrt values
- `--fresh` archives checkpoint only; it does not delete keyring entries

## E2E live

- Local: `E2E_LIVE_MODE=fixture make e2e-live` (Go fixture; no GitLab registration token)
- GitHub: manual **e2e-live** workflow (`workflow_dispatch`)
- Refuses to seed if a gitlab-runner service already exists unless `E2E_LIVE_ALLOW_EXISTING=1`
- `repair-prod` observe path: `E2E_LIVE_ALLOW_PROD=1` (not used on GHA)
- Re-run after `gitlab-runner` upgrades or changes under `pkg/service`, `pkg/inventory`, `pkg/repair`, or `pkg/cleanup`
