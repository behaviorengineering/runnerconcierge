# runnerconcierge-operator

Operate the runnerconcierge GitLab runner setup CLI.

## MUST

- Run `runnerconcierge doctor` before setup on a new host.
- Use login user for Windows service install (never LocalSystem).
- Pass tags only via GitLab `POST /user/runners`, not `gitlab-runner register`.
- Store PAT in keyring or env; never commit `glrt` tokens.

## Commands

- `runnerconcierge` or `runnerconcierge setup`: wizard
- `runnerconcierge init`: seed config
- `runnerconcierge verify`: service status
