# runnerconcierge

Interactive Go CLI that installs, registers, and starts GitLab self-hosted runners on macOS and Windows under the login user (not LocalSystem).

Module: `github.com/behaviorengineering/runnerconcierge`

## Quick start

```bash
make hooks-install
make build
./bin/runnerconcierge init
./bin/runnerconcierge        # setup wizard
./bin/runnerconcierge doctor
./bin/runnerconcierge status
```

Day-2 inventory (manual installs, wrong service user):

```bash
./bin/runnerconcierge status
./bin/runnerconcierge repair-service --runner-config ~/.gitlab-runner/config.toml
```

Live E2E (maintainers; macOS/Windows):

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
