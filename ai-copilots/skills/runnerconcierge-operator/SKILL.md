# runnerconcierge-operator

Operate the runnerconcierge GitLab runner setup CLI.

## MUST

- Run `runnerconcierge doctor` before setup on a new host.
- Run `runnerconcierge status` to inventory existing runners and smells (wrong user, LocalSystem, system config).
- Use login user for Windows service install (never LocalSystem).
- Pass tags only via GitLab `POST /user/runners`, not `gitlab-runner register`.
- Store PAT in keyring or env; never commit `glrt` tokens.
- Re-run `make e2e-live` after `gitlab-runner` upgrades or changes under `pkg/service`, `pkg/inventory`, or `pkg/repair`.

## Commands

- `runnerconcierge`: agent operating guide (no setup)
- `runnerconcierge setup`: wizard (`--repo group/project` required to create a project runner)
- `runnerconcierge init`: seed config
- `runnerconcierge doctor`: setup preflight (labeled sections; `--json` for agents; `--docker` to require Docker)
- `runnerconcierge verify`: one-line service status (GitLab native install or brew services on macOS)
- `runnerconcierge status`: full machine inventory and smells (`--json` for agents)
- `runnerconcierge repair-service --runner-config PATH`: rebind service to login user (confirm or `--yes`; Windows password via prompt, `--windows-password`, or `RUNNERCONCIERGE_WINDOWS_PASSWORD`)

## E2E live

- Local: `E2E_LIVE_MODE=fixture make e2e-live` (Go fixture in `internal/e2elive/fixture`; no GitLab registration token).
- `E2E_LIVE_REQUIRE=1`: missing `gitlab-runner` or fixture skip is a **failed** run (use in CI).
- Refuses to seed if a gitlab-runner service already exists unless `E2E_LIVE_ALLOW_EXISTING=1`.
- `repair-prod` observe path: `E2E_LIVE_ALLOW_PROD=1` (not used on GHA; never set in **e2e-host**).

### Host sandbox (public repo)

This repository is **public**. A self-hosted runner that picks up fork `pull_request` jobs runs **untrusted** code with service-install rights.

- MUST NOT register the e2e runner for generic `pull_request` or `pull_request_target` workflows.
- MUST NOT attach the Polypus / homelab **deploy** GitLab runner to this GitHub e2e pool.
- Use workflow **e2e-host** only (`workflow_dispatch`, or `workflow_run` after **CI** on `main`). Job uses Environment **e2e-host** (required reviewers) and `runs-on: [self-hosted, e2e, macos]` or `[self-hosted, e2e, windows]`.
- Register the GitHub runner under a **dedicated OS user** (for example `e2e-runner`), not the login user that holds production keyring or `~/.gitlab-runner` for homelab runners.
- After registration, remove the default **self-hosted** label if GitHub added it, so other workflows cannot accidentally target this machine.
- Repo Settings → Actions: block public fork workflows from using self-hosted runners where your org policy allows.
- Job env: `E2E_LIVE_MODE=fixture`, `E2E_LIVE_REQUIRE=1` only. No repo secrets, no `E2E_LIVE_ALLOW_PROD`.
- Fixture services use prefix `runnerconcierge-e2e-`; production service names must be unchanged after the job (harness snapshots).

One-time forge setup:

1. Create GitHub Environment `e2e-host` with required reviewers (no secrets).
2. Install `gitlab-runner` on PATH for the dedicated user.
3. Register one macOS and one Windows runner with labels `e2e` and `macos` or `windows` (not untagged).
4. Run **e2e-host** via `workflow_dispatch` and approve the Environment; confirm dispose leaves no `runnerconcierge-e2e-*` plists/services.

Optional: **e2e-live** workflow on GitHub-hosted `macos-latest` / `windows-latest` for maintainers without homelab runners.
