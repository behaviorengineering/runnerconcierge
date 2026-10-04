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

- `runnerconcierge` or `runnerconcierge setup`: wizard
- `runnerconcierge init`: seed config
- `runnerconcierge doctor`: setup preflight
- `runnerconcierge verify`: one-line service status (scripts)
- `runnerconcierge status`: full machine inventory and smells (`--json` for agents)
- `runnerconcierge repair-service --runner-config PATH`: rebind service to login user (confirm or `--yes`; Windows password via prompt, `--windows-password`, or `RUNNERCONCIERGE_WINDOWS_PASSWORD`)

## E2E live

- `make e2e-live` on macOS and Windows before releases that touch status/repair.
- Modes: `E2E_LIVE_MODE=fixture` (default), `observe`, `repair-prod` (requires `E2E_LIVE_ALLOW_PROD=1`).
- Windows fixture mode needs an elevated session.
