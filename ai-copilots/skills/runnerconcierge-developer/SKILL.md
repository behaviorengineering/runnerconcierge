# runnerconcierge-developer

Extend runnerconcierge packages and wizard stages.

## Layout

- `cmd/runnerconcierge`: olly init, cli entry
- `internal/cli`, `internal/wizard`, `internal/config`
- `pkg/detect`, `pkg/install`, `pkg/gitlabrunner`, `pkg/service`, `pkg/inventory`, `pkg/repair`, `pkg/state`, `pkg/redact`, `pkg/preset`, `pkg/errdefs`
- `internal/e2elive` (`e2e_live` build tag)

## Tests

`make test` and `make smoke`. GitLab API: use `httptest` in `pkg/gitlabrunner`.
