# runnerconcierge ai-copilots

Operator and developer pack for this module. Canonical source lives here.

**Load order**

1. [skills/runnerconcierge-operator/SKILL.md](skills/runnerconcierge-operator/SKILL.md)
2. [skills/runnerconcierge-developer/SKILL.md](skills/runnerconcierge-developer/SKILL.md) when changing Go or CLI wiring
3. [skills/gitlab-runner-cli/SKILL.md](skills/gitlab-runner-cli/SKILL.md) when changing `gitlab-runner` argv or flag compat tests
4. [BOOTSTRAP.md](BOOTSTRAP.md) before the first commit in a clone or worktree

**Minimal prompt**

> Wire runnerconcierge ai-copilots using BOOTSTRAP.md
