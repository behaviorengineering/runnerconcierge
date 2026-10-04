# Agents

GitLab self-hosted runner setup CLI for macOS and Windows. Humans read [README.md](README.md).

**Load before you operate or change this module:**

1. [ai-copilots/README.md](ai-copilots/README.md)
2. [ai-copilots/skills/runnerconcierge-operator/SKILL.md](ai-copilots/skills/runnerconcierge-operator/SKILL.md) (doctor, status, cleanup, runners gitlab list/setup)
3. [ai-copilots/skills/runnerconcierge-developer/SKILL.md](ai-copilots/skills/runnerconcierge-developer/SKILL.md) (when changing packages, CLI, or tests)
4. [ai-copilots/skills/gitlab-runner-cli/SKILL.md](ai-copilots/skills/gitlab-runner-cli/SKILL.md) (when changing `gitlab-runner` argv, flags, or compat tests)
5. Execute [ai-copilots/BOOTSTRAP.md](ai-copilots/BOOTSTRAP.md) in this clone before the first commit.

Resolve this module when it is only a Go dependency:

```bash
go list -m -f '{{.Dir}}' github.com/behaviorengineering/runnerconcierge
```

MUST keep host IDE links pointing at this module's `ai-copilots/` tree. MUST NOT copy skill bodies into a host unless links fail and the user approves copy fallback.
