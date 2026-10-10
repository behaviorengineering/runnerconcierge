# runnerconcierge ai-copilots

Operator and developer pack for this module. Canonical source lives here.

**Load order**

1. [skills/runnerconcierge-operator/SKILL.md](skills/runnerconcierge-operator/SKILL.md)
2. [skills/runnerconcierge-developer/SKILL.md](skills/runnerconcierge-developer/SKILL.md) when changing Go or CLI wiring
3. [skills/gitlab-runner-cli/SKILL.md](skills/gitlab-runner-cli/SKILL.md) when changing `gitlab-runner` argv or flag compat tests

IF IDE discovery links are missing for your agent product, run [BOOTSTRAP.md](BOOTSTRAP.md) in wire-only mode before you operate (see [AGENTS.md](../AGENTS.md)). First commit in a clone: `go tool task hooks-install` plus wire.

**Minimal prompt**

> Wire runnerconcierge ai-copilots using BOOTSTRAP.md
