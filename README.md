# runnerconcierge

Interactive Go CLI that installs, registers, and starts GitLab self-hosted runners on macOS and Windows under the login user (not LocalSystem).

Module: `github.com/behaviorengineering/runnerconcierge`

## Quick start

```bash
make hooks-install
make build
./bin/runnerconcierge init
./bin/runnerconcierge        # agent operating guide (no setup)
./bin/runnerconcierge setup  # setup wizard (interactive or --non-interactive --yes)
./bin/runnerconcierge doctor
./bin/runnerconcierge status
```

Day-2 inventory (manual installs, wrong service user):

```bash
./bin/runnerconcierge status
./bin/runnerconcierge repair-service --runner-config ~/.gitlab-runner/config.toml
```

Live E2E (maintainers; macOS/Windows only; not part of Ubuntu PR CI):

Local-first (fixture install → status smell → repair → dispose; **no GitLab register**):

```bash
make build
E2E_LIVE_MODE=fixture make e2e-live
```

Fail if the test would skip (CI-style):

```bash
E2E_LIVE_MODE=fixture E2E_LIVE_REQUIRE=1 make e2e-live
```

On a machine that already has a gitlab-runner service, the fixture uses an **isolated service name** (`runnerconcierge-e2e-*`) so your main runner is not removed. On macOS with an existing runner, the seed and repair steps need **sudo** for the temporary system LaunchDaemon. In Cursor’s terminal, `sudo -v` often does not reach `go test`; set `E2E_LIVE_SUDO_PASSWORD` (or `RUNNERCONCIERGE_SUDO_PASSWORD`) in that shell for the run (local only; do not commit). Optional traces: `E2E_LIVE_ARTIFACT_DIR=/tmp/e2e-artifacts`.

GitHub Actions:

- **e2e-host** (recommended for homelab): `workflow_dispatch` or after **CI** succeeds on `main`. Runs on self-hosted runners labeled `e2e` + `macos` or `e2e` + `windows`, GitHub Environment `e2e-host` (required reviewers). Trusted repo SHA only; no `pull_request` trigger. See operator skill for runner registration.
- **e2e-live**: optional manual run on GitHub-hosted `windows-latest` / `macos-latest` (same `make e2e-live`).

All install/repair logic is Go in `internal/e2elive/fixture`.

Observe-only inventory (no fixture):

```bash
make e2e-live
```

Automation:

```bash
runnerconcierge setup --non-interactive --yes \
  --repo group/project --tag-list my-tag --pat "$GITLAB_TOKEN"
```

## Forge setup

See [docs/FORGE.md](docs/FORGE.md) (prepare-go-forge, branch protection, secret scan, required checks).

## License

Apache-2.0
