# GitHub forge setup

After the first push to `github.com/behaviorengineering/runnerconcierge`:

1. Create the public repository (if not already created).
2. Run **prepare-go-forge** for GitHub (`github-prepare-go-forge.sh behaviorengineering runnerconcierge`).
3. Confirm required checks: **CI** (`quality` on ubuntu), **gitleaks** (`secret-scan.yml`). Multi-platform binaries come from GoReleaser on tag release, not the PR matrix.
4. Protect `main` (ruleset or branch protection) so merges require those checks.
5. Tag `v0.1.0` manually once Windows and macOS smoke pass, or let **auto-patch-release** publish after merge.

Release binaries: push a `v*` tag or merge to `main` for auto-patch (see `.github/workflows/release.yml` and `auto-patch-release.yml`).
