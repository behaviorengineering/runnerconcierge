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
```

Automation:

```bash
runnerconcierge setup --non-interactive --yes \
  --repo group/project --tag-list my-tag --pat "$GITLAB_TOKEN"
```

## Forge setup

After pushing to GitHub, run the prepare-go-forge script for `behaviorengineering/runnerconcierge` (branch protection, secret scan, required checks).

## License

Apache-2.0
