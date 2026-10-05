# BOOTSTRAP — runnerconcierge ai-copilots

**Audience:** Any AI agent (Cursor, GitHub Copilot, Claude Code, Codex) in a workspace that depends on or checks out this module.

**Goal:** Wire host IDE discovery to canonical content under `ai-copilots/`. Optionally refresh content. MUST NOT copy skill bodies unless symlinks or junctions fail and the user approves copy fallback.

**Module path:** `github.com/behaviorengineering/runnerconcierge`

---

## When to run

| Trigger | Mode | Phases |
|---------|------|--------|
| Missing IDE discovery links (operate) | **Wire only** | 0 → 2 → 3 → 4 |
| User asks to refresh harness content | **Refresh content + wire** | 0 → 1 → 2 → 3 → 4 |

IF `.cursor/skills/runnerconcierge-operator` or `.github/skills/runnerconcierge-operator` is missing, run wire-only **before** you operate.

First commit in this clone: run `make hooks-install` (Lefthook) after wire verification in phase 4. That is separate from operate-time wire-if-missing.

---

## Phase 0 — Resolve module root

From a Go module that requires `github.com/behaviorengineering/runnerconcierge` (or this checkout):

```bash
MOD="$(go list -m -f '{{.Dir}}' github.com/behaviorengineering/runnerconcierge)"
test -d "$MOD/ai-copilots" || { echo "missing ai-copilots under $MOD"; exit 1; }
echo "Module Dir: $MOD"
```

If `go list` is unavailable, use a known nested checkout path only when it clearly contains `ai-copilots/`. MUST NOT invent a path.

Re-run wire after module version bumps (cache Dir can change).

---

## Phase 1 — Refresh content (optional)

Edit only files under `$MOD/ai-copilots/`. Load author-ai-copilots + agent-smith when authoring skills.

Target tree:

```text
ai-copilots/
  README.md
  BOOTSTRAP.md
  skills/runnerconcierge-operator/SKILL.md
  skills/runnerconcierge-developer/SKILL.md
  skills/gitlab-runner-cli/SKILL.md
```

---

## Phase 2 — Ask IDE and OS if unknown

1. IDE: Cursor, GitHub Copilot, Claude Code, Codex
2. OS: macOS/Linux symlink vs Windows junction/copy
3. Workspace: library alone vs nested under a parent vs dependency-only

---

## Phase 3 — Wire discovery

Canonical sources:

| Artifact | Path under `$MOD` |
|----------|-------------------|
| Skill tree | `ai-copilots/skills/runnerconcierge-operator/` |
| Skill tree | `ai-copilots/skills/runnerconcierge-developer/` |
| Skill tree | `ai-copilots/skills/gitlab-runner-cli/` |

Discovery paths:

| IDE | Agents | Skills |
|-----|--------|--------|
| Cursor | `.cursor/agents/*.md` | `.cursor/skills/**/SKILL.md` |
| GitHub Copilot | `.github/agents/*.agent.md` | `.github/skills/**/SKILL.md` |
| Claude Code | `.claude/agents/*.md` | `.claude/skills/**/SKILL.md` |
| Codex | `.codex/agents/*.md` | `.codex/skills/**/SKILL.md` |

**Cursor example (macOS/Linux)** from the **host workspace root**:

```bash
MOD="$(go list -m -f '{{.Dir}}' github.com/behaviorengineering/runnerconcierge)"
mkdir -p .cursor/skills
ln -snf "$MOD/ai-copilots/skills/runnerconcierge-operator" .cursor/skills/runnerconcierge-operator
ln -snf "$MOD/ai-copilots/skills/runnerconcierge-developer" .cursor/skills/runnerconcierge-developer
ln -snf "$MOD/ai-copilots/skills/gitlab-runner-cli" .cursor/skills/gitlab-runner-cli
```

When the workspace root is this library itself, relative links are fine:

```bash
ln -snf ../ai-copilots/skills/runnerconcierge-operator .cursor/skills/runnerconcierge-operator
ln -snf ../ai-copilots/skills/runnerconcierge-developer .cursor/skills/runnerconcierge-developer
ln -snf ../ai-copilots/skills/gitlab-runner-cli .cursor/skills/gitlab-runner-cli
```

**Windows:** prefer junction or developer-mode symlink; copy fallback only with user approval.

**Idempotency:** skip if the link already resolves to the canonical path; ask before overwriting stale copies.

---

## Phase 4 — Verify

```bash
MOD="$(go list -m -f '{{.Dir}}' github.com/behaviorengineering/runnerconcierge)"
test -f "$MOD/ai-copilots/BOOTSTRAP.md"
test -f "$MOD/ai-copilots/skills/runnerconcierge-operator/SKILL.md"
test -f "$MOD/ai-copilots/skills/runnerconcierge-developer/SKILL.md"
test -f "$MOD/ai-copilots/skills/gitlab-runner-cli/SKILL.md"
make -C "$MOD" hooks-install
```

Ask the user before committing host wiring (`.cursor/`, `.github/`, etc.).
