# runnerconcierge

Interactive Go CLI that installs, registers, and starts GitLab self-hosted runners on macOS and Windows under the login user (not LocalSystem).

Module: `github.com/behaviorengineering/runnerconcierge`

## Quick start

```bash
go tool task hooks-install
go tool task          # targets + GitLab workflow
go tool task init
go tool task build; bin/runnerconcierge runners gitlab setup  # wizard; on macOS/Windows also installs docker cleanup schedule
go tool task build; bin/runnerconcierge runners gitlab list
./bin/runnerconcierge doctor
./bin/runnerconcierge status
```

Day-2 inventory (manual installs, wrong service user):

```bash
./bin/runnerconcierge status
./bin/runnerconcierge repair-service --runner-config ~/.gitlab-runner/config.toml
./bin/runnerconcierge runners gitlab list
./bin/runnerconcierge runners gitlab list --json
./bin/runnerconcierge runners gitlab list --name "$(hostname)" --action remove --yes
./bin/runnerconcierge cleanup
./bin/runnerconcierge cleanup install --yes
```

Live E2E (maintainers; macOS/Windows only; not part of Ubuntu PR CI):

Local-first (fixture install → status smell → repair → dispose; **no GitLab register**):

```bash
go tool task build
E2E_LIVE_MODE=fixture go tool task e2e-live
```

Fail if the test would skip (CI-style):

```bash
E2E_LIVE_MODE=fixture E2E_LIVE_REQUIRE=1 go tool task e2e-live
```

On a machine that already has a gitlab-runner service, the fixture uses an **isolated service name** (`runnerconcierge-e2e-*`) so your main runner is not removed. On macOS with an existing runner, the seed and repair steps need **sudo** for the temporary system LaunchDaemon. In Cursor’s terminal, `sudo -v` often does not reach `go test`; set `E2E_LIVE_SUDO_PASSWORD` (or `RUNNERCONCIERGE_SUDO_PASSWORD`) in that shell for the run (local only; do not commit). Optional traces: `E2E_LIVE_ARTIFACT_DIR=/tmp/e2e-artifacts`.

GitHub Actions: run the **e2e-live** workflow manually (`workflow_dispatch`) on `windows-latest` / `macos-latest`. The workflow only runs `go tool task e2e-live`; all install/repair logic is Go in `internal/e2elive/fixture`.

Observe-only inventory (no fixture):

```bash
go tool task e2e-live
```

Automation:

```bash
runnerconcierge runners gitlab setup --non-interactive --yes \
  --repo group/project --tag-list my-tag --pat "$GITLAB_TOKEN"
```

Resume after a failed register: `state.json` keeps runner id, tags, executor, and project/group paths (no glrt). Setup uses `glab auth login` and API create/reset for glrt. The keyring stores `GITLAB_RUNNER_TOKEN_<identity>` where identity is `parent-tag-hostname` (same string as GitLab description and `config.toml` name). Keyring glrt is reused only when `state.json` has a `runner_id`; a new setup mints a token unless you pass `--token`. If register reports the token is not valid, setup resets or creates once and retries. Legacy `GITLAB_RUNNER_TOKEN_<id>` / `<tag>` entries are read as fallback only while resuming that runner id.

## Forge setup

See [docs/FORGE.md](docs/FORGE.md) (prepare-go-forge, branch protection, secret scan, required checks).

## License

Apache-2.0
