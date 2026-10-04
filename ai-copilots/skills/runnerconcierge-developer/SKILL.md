# runnerconcierge-developer

Extend runnerconcierge packages and wizard stages.

## Layout

- `cmd/runnerconcierge`: olly init, cli entry
- `internal/cli`, `internal/wizard`, `internal/config`
- `pkg/detect`, `pkg/install`, `pkg/gitlabrunner`, `pkg/service`, `pkg/inventory`, `pkg/repair`, `pkg/state`, `pkg/redact`, `pkg/preset`, `pkg/errdefs`
- `internal/e2elive` (`e2e_live` build tag), `internal/e2elive/fixture` (Go-driven live fixture)

## Live e2e env

| Variable | Purpose |
|----------|---------|
| `E2E_LIVE_MODE=fixture` | Run `TestFixtureLifecycle` |
| `E2E_LIVE_REQUIRE=1` | Skip → fatal on darwin/windows |
| `E2E_LIVE_ALLOW_EXISTING=1` | Allow seed when gitlab-runner services already exist |
| `E2E_LIVE_SET_WINDOWS_PASSWORD=1` | Allow `net user` password reset for repair (also on `GITHUB_ACTIONS`) |
| `RUNNERCONCIERGE_WINDOWS_PASSWORD` | Windows service install password for repair |
| `E2E_LIVE_ARTIFACT_DIR` | Write inventory JSON traces |

Workflow: `.github/workflows/e2e-live.yml` (`make e2e-live` only).

CLI: bare invoke prints agent guide only; `setup` runs the wizard. Domain errors use `pkg/errdefs` (`Error()` omits cause argv; `FormatCLI` for stderr).

## Tests

`make test` and `make smoke`. GitLab API: use `httptest` in `pkg/gitlabrunner`.
