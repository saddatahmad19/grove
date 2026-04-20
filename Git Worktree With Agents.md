# Multi-Agent Git Worktree Workflow
## Production-Grade JavaScript Monorepo with Parallel AI Coding Agents

> **Environment:** Kubuntu + Niri + Zellij + Neovim + pnpm | Local-first, cloud inference | Human-in-the-loop

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Repository & Bare Clone Setup](#2-repository--bare-clone-setup)
3. [Directory Structure & Workspace Layout](#3-directory-structure--workspace-layout)
4. [Git Worktree Strategy](#4-git-worktree-strategy)
5. [Agent Role Architecture](#5-agent-role-architecture)
6. [Task Routing System](#6-task-routing-system)
7. [Context-Sharing & State Management](#7-context-sharing--state-management)
8. [CLI Automation Scripts](#8-cli-automation-scripts)
9. [Human-in-the-Loop Checkpoints](#9-human-in-the-loop-checkpoints)
10. [Code Quality & Security Workflows](#10-code-quality--security-workflows)
11. [Zellij Session Architecture](#11-zellij-session-architecture)
12. [Failure Handling Strategies](#12-failure-handling-strategies)
13. [Performance & Resource Optimization](#13-performance--resource-optimization)
14. [Quick Reference Card](#14-quick-reference-card)

---

## 1. System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                     MONOREPO BARE CLONE (.git/)                     │
│                    /projects/myapp.git  (canonical)                 │
└──────────────┬──────────────┬───────────────┬───────────────────────┘
               │              │               │
    ┌──────────▼───┐  ┌───────▼──────┐  ┌────▼──────────┐
    │  wt/main     │  │  wt/feature/ │  │  wt/bugfix/   │
    │  (read-only  │  │  <slug>      │  │  <slug>       │   ...more
    │   reference) │  │              │  │               │
    └──────────────┘  └──────┬───────┘  └───────┬───────┘
                             │                  │
                    ┌────────▼──────────────────▼────────┐
                    │         AGENT DISPATCH LAYER        │
                    │  hermes → claude-code → codex →    │
                    │  opencode  (routed by task type)   │
                    └────────────────┬───────────────────┘
                                     │
                    ┌────────────────▼───────────────────┐
                    │       SHARED CONTEXT STORE          │
                    │  .agent/tasks/  .agent/memory/      │
                    │  .agent/logs/   .agent/state.json   │
                    └────────────────────────────────────┘
```

### Core Principles

- **One worktree per concern** — features, bugfixes, experiments, and audits each live in isolated filesystem trees
- **Agents are stateless executors** — all state lives in the shared context store, not inside agent sessions
- **Human gates are non-negotiable** — no code enters `main` without a human approval checkpoint
- **pnpm workspaces** persist across worktrees via a single shared store at `~/.pnpm-store`
- **Zellij replaces tmux** — each worktree maps to a named Zellij session with a standardized layout

---

## 2. Repository & Bare Clone Setup

Using a **bare clone** as the canonical source lets worktrees share a single `.git` object store without the performance hit of separate full clones.

```bash
# One-time setup — creates the bare repo
mkdir -p ~/projects
git clone --bare git@github.com:yourorg/myapp.git ~/projects/myapp.git

# Set the fetchspec so `git fetch` works from inside worktrees
git -C ~/projects/myapp.git config remote.origin.fetch '+refs/heads/*:refs/remotes/origin/*'

# Create a permanent `main` worktree as a read-only reference
git -C ~/projects/myapp.git worktree add ~/projects/myapp/main main
```

> **Why bare?**
> A normal clone keeps a checked-out working tree at the root. With a bare repo, the object store *is* the root — worktrees are pure working trees with no duplication of `.git` data.

---

## 3. Directory Structure & Workspace Layout

```
~/projects/
├── myapp.git/                   # Bare repo (canonical object store)
│
├── myapp/
│   ├── main/                    # Permanent reference worktree (read-only by convention)
│   │
│   ├── wt/
│   │   ├── feat-auth-mfa/       # Feature worktree
│   │   ├── feat-billing-v2/     # Feature worktree
│   │   ├── fix-rls-policy/      # Bugfix worktree
│   │   ├── exp-edge-runtime/    # Experiment worktree
│   │   └── audit-deps-q2/       # Security audit worktree
│   │
│   └── .agent/                  # Shared context store (NOT inside any worktree)
│       ├── state.json           # Global task registry
│       ├── tasks/
│       │   ├── feat-auth-mfa.yaml
│       │   └── fix-rls-policy.yaml
│       ├── memory/
│       │   ├── codebase.md      # Persistent architectural notes
│       │   ├── conventions.md   # Coding standards digest
│       │   └── decisions.md     # ADR-style decision log
│       ├── logs/
│       │   ├── hermes.log
│       │   ├── claude-code.log
│       │   ├── codex.log
│       │   └── opencode.log
│       └── handoffs/            # Inter-agent handoff packets
│           └── feat-auth-mfa-codex→claude.json
│
~/.config/myapp-agents/
│   ├── routing.yaml             # Task routing rules
│   ├── agents.yaml              # Agent configuration & API keys
│   └── hooks/                   # Git hooks shared across worktrees
│       ├── pre-commit
│       └── pre-push
```

### pnpm Workspace Consideration

All worktrees share the global pnpm store. Each worktree has its own `node_modules` but avoids re-downloading packages:

```bash
# In each new worktree
pnpm install --prefer-offline
```

To prevent `pnpm` from locking across worktrees, ensure each worktree has its own `.npmrc` lockfile reference — the bare repo approach handles this automatically since each worktree is an independent filesystem tree.

---

## 4. Git Worktree Strategy

### 4.1 Naming Conventions

```
wt/<type>-<scope>-<short-description>

Types:
  feat     → New feature
  fix      → Bug fix
  exp      → Experiment / spike (disposable)
  audit    → Security or dependency audit
  refactor → Structural improvement, no feature change
  chore    → Tooling, CI, config changes
```

**Examples:**
```
wt/feat-auth-mfa
wt/fix-rls-policy-bypass
wt/exp-edge-runtime-streaming
wt/audit-deps-q2-2025
wt/refactor-api-layer-types
```

### 4.2 Branching Model

```
main
 └── develop                      ← integration branch
      ├── feat/auth-mfa            ← maps to wt/feat-auth-mfa
      ├── feat/billing-v2          ← maps to wt/feat-billing-v2
      ├── fix/rls-policy-bypass    ← maps to wt/fix-rls-policy-bypass
      └── exp/edge-runtime         ← maps to wt/exp-edge-runtime (never merges to main directly)
```

**Rules:**
- `main` is **merge-only** — no direct pushes, ever
- `develop` is rebased from `main` weekly (automated, human-confirmed)
- Feature/fix branches rebase from `develop`, not `main`
- Experiment branches are **ephemeral** — they never merge to `develop` or `main`; findings are extracted as conventional commits to a new branch
- All merges to `main` go through a PR with required human approval

### 4.3 Lifecycle Management

```
Created → Active → Review → Merged/Abandoned → Pruned
```

- **Active:** Agent is assigned and working
- **Review:** Human checkpoint triggered; agent paused
- **Merged:** PR merged; worktree can be destroyed
- **Abandoned:** Experiment yielded no results; branch deleted, worktree destroyed
- **Pruned:** `git worktree prune` removes stale worktree references

---

## 5. Agent Role Architecture

### 5.1 Agent Profiles

```yaml
# ~/.config/myapp-agents/agents.yaml

agents:
  hermes:
    role: orchestrator
    model: gpt-5.4-pro          # via Hermes Agent API
    capabilities:
      - task_decomposition
      - cross-agent_planning
      - codebase_reasoning
      - blocking_decision_making
    context_budget: 128k
    timeout_seconds: 300
    cost_tier: high              # Only invoked for planning, not implementation

  claude-code:
    role: analyst_refactorer
    model: claude-sonnet-4-6     # via Claude Code CLI
    capabilities:
      - deep_code_analysis
      - safety_critical_logic
      - refactoring
      - security_review
      - type_system_reasoning
    context_budget: 200k
    timeout_seconds: 600
    cost_tier: high

  codex:
    role: implementer
    model: gpt-5.3-codex         # via OpenAI Codex API
    capabilities:
      - fast_code_generation
      - scaffolding
      - boilerplate
      - test_generation
      - routine_implementation
    context_budget: 32k
    timeout_seconds: 120
    cost_tier: medium

  opencode:
    role: auxiliary
    model: openrouter_auto        # via OpenRouter (model-agnostic)
    capabilities:
      - long_tail_experiments
      - overflow_workloads
      - auxiliary_research
      - documentation_generation
      - dependency_analysis
    context_budget: 64k
    timeout_seconds: 240
    cost_tier: low
```

### 5.2 Agent Responsibilities in Detail

#### Hermes — The Orchestrator
- Receives a raw task description from the human
- Reads `.agent/memory/codebase.md` and `.agent/memory/conventions.md` for context
- Decomposes the task into subtasks with dependency ordering
- Assigns each subtask to the appropriate agent
- Writes the task plan to `.agent/tasks/<slug>.yaml`
- Does **not** write source code directly

```
Input:  "Add MFA support to the auth flow"
Output: .agent/tasks/feat-auth-mfa.yaml with:
  - subtask 1: scaffold MFA schema migrations (→ Codex)
  - subtask 2: implement TOTP verification logic (→ Claude Code)
  - subtask 3: build UI components for MFA enrollment (→ Codex)
  - subtask 4: security review of auth state transitions (→ Claude Code)
  - subtask 5: generate unit + integration tests (→ Codex)
  - HUMAN GATE: review before merge
```

#### Claude Code — The Analyst
- Works inside a worktree via the `claude` CLI
- Specializes in logic that must be **correct, not just functional**
- Handles: auth flows, RLS policies, data validation schemas (Zod), type narrowing, error boundaries
- Produces annotated diffs with reasoning comments
- Always writes to `.agent/logs/claude-code.log` with structured output
- Can request a Hermes re-plan if scope expands unexpectedly

#### Codex — The Implementer
- Fast, token-efficient code generation via the Codex API
- Handles: React components, API route scaffolding, test boilerplate, migration files, utility functions
- Works from a Hermes-generated spec; does not do open-ended reasoning
- Output goes into the worktree; Claude Code reviews before the human gate

#### OpenCode — The Auxiliary
- OpenRouter-backed, model-agnostic (routes to the cheapest capable model automatically)
- Used for: dependency research, README generation, changelog drafting, exploratory spikes, documentation
- Also acts as **overflow** when Claude Code or Codex API latency exceeds thresholds
- Experiment worktrees (`wt/exp-*`) are exclusively owned by OpenCode

---

## 6. Task Routing System

### 6.1 Routing Rules

```yaml
# ~/.config/myapp-agents/routing.yaml

routes:
  # Primary routing by task type
  - match:
      type: [scaffold, generate, boilerplate, migration, test_generation]
    agent: codex
    fallback: opencode

  - match:
      type: [refactor, security_review, auth_logic, type_safety, rls_policy]
    agent: claude-code
    fallback: opencode
    requires_human_review: true

  - match:
      type: [planning, decomposition, cross_cutting, architecture]
    agent: hermes
    fallback: claude-code   # Claude Code can plan if Hermes is unavailable

  - match:
      type: [experiment, research, documentation, changelog, overflow]
    agent: opencode
    fallback: codex

  # Escalation rules
  - match:
      flags: [security_critical]
    agent: claude-code
    requires_human_review: true
    blocks_merge: true

  - match:
      flags: [touches_auth, touches_billing, touches_phi]
    agent: claude-code
    requires_human_review: true

  # Fallback chain (when primary agent API is degraded)
  fallback_chain:
    hermes:    [claude-code, opencode]
    claude-code: [opencode, hermes]
    codex:     [opencode, claude-code]
    opencode:  [codex, claude-code]
```

### 6.2 Routing Decision Flow

```
New Task Arrives
       │
       ▼
   Parse task metadata
   (type, flags, scope)
       │
       ├─► security_critical or touches_phi?
       │         │
       │         └─► Force route → claude-code + set human_gate=required
       │
       ├─► task.type in scaffold/generate?
       │         │
       │         └─► Route → codex
       │                   │
       │                   └─► codex API latency > 10s? → fallback → opencode
       │
       ├─► task.type in refactor/security_review?
       │         │
       │         └─► Route → claude-code
       │
       └─► task.type in experiment/docs?
                 │
                 └─► Route → opencode
```

### 6.3 Arbitration Logic

When two agents produce conflicting implementations (e.g., Codex scaffolds a function that Claude Code's analysis contradicts):

1. **Conflict detected** by diffing handoff packets in `.agent/handoffs/`
2. Hermes is invoked as **arbiter** — it reads both outputs and the original task spec
3. Hermes produces a resolution plan with explicit line-level justification
4. If Hermes cannot resolve (e.g., ambiguous business requirement), a **human escalation** is triggered immediately
5. The conflicting worktree is **frozen** (no further agent writes) until resolution

```bash
# Freeze a worktree pending arbitration
wf freeze wt/feat-auth-mfa "conflict: codex vs claude-code on token refresh logic"
```

---

## 7. Context-Sharing & State Management

### 7.1 Global State File

```json
// .agent/state.json
{
  "schema_version": "1.2",
  "updated_at": "2025-04-19T14:32:00Z",
  "worktrees": {
    "feat-auth-mfa": {
      "branch": "feat/auth-mfa",
      "status": "active",
      "assigned_agent": "claude-code",
      "task_file": ".agent/tasks/feat-auth-mfa.yaml",
      "human_gate": "pending",
      "last_commit": "a3f9c12",
      "api_calls_today": 47,
      "created_at": "2025-04-18T09:00:00Z"
    },
    "fix-rls-policy": {
      "branch": "fix/rls-policy-bypass",
      "status": "review",
      "assigned_agent": "claude-code",
      "task_file": ".agent/tasks/fix-rls-policy.yaml",
      "human_gate": "required",
      "flags": ["security_critical"],
      "last_commit": "b7d1e45"
    }
  },
  "api_health": {
    "hermes": "healthy",
    "claude-code": "healthy",
    "codex": "degraded",
    "opencode": "healthy"
  }
}
```

### 7.2 Task File Schema

```yaml
# .agent/tasks/feat-auth-mfa.yaml

task:
  id: feat-auth-mfa
  title: "Add MFA support to auth flow"
  type: feature
  priority: high
  flags: [touches_auth, security_critical]
  worktree: wt/feat-auth-mfa
  branch: feat/auth-mfa
  base_branch: develop

plan:
  created_by: hermes
  created_at: "2025-04-18T09:15:00Z"
  subtasks:
    - id: st-001
      title: "Scaffold MFA schema migration"
      agent: codex
      status: completed
      commit: a1b2c3d
      output_file: "supabase/migrations/20250418_mfa.sql"

    - id: st-002
      title: "Implement TOTP verification logic"
      agent: claude-code
      status: in_progress
      depends_on: [st-001]
      context_files:
        - src/lib/auth/session.ts
        - src/lib/auth/middleware.ts

    - id: st-003
      title: "MFA enrollment UI components"
      agent: codex
      status: blocked
      depends_on: [st-002]

    - id: st-004
      title: "Security review of auth state transitions"
      agent: claude-code
      status: pending
      depends_on: [st-002, st-003]
      human_gate: required_before_next

    - id: st-005
      title: "Generate unit + integration tests"
      agent: codex
      status: pending
      depends_on: [st-004]

human_gates:
  - after: st-004
    type: security_review
    approver: human
    status: pending
    notes: ""

context:
  memory_refs:
    - .agent/memory/codebase.md#auth-section
    - .agent/memory/conventions.md#error-handling
  relevant_files:
    - src/lib/auth/
    - src/app/(auth)/
    - supabase/migrations/
```

### 7.3 Memory Files

The `.agent/memory/` directory is the persistent brain shared by all agents:

```markdown
<!-- .agent/memory/codebase.md -->

# Codebase Memory

## Architecture
- Next.js App Router, TypeScript strict mode
- Supabase for DB + Auth; Clerk for multi-tenant subdomain auth
- RTK Query for server state; Zod for all runtime validation
- pnpm workspaces: apps/web, packages/ui, packages/db

## Auth Layer
- Session management via Clerk JWTs + Supabase RLS
- All auth mutations go through src/lib/auth/session.ts
- Never bypass RLS — use service role only in edge functions
- MFA state is tracked in users.mfa_enabled (boolean) + auth.mfa_factors

## Known Footguns
- RTK Query store must be instantiated with useRef (see: #rtk-hydration-bug)
- Supabase realtime subscriptions leak if not cleaned up in useEffect returns
- Clerk webhook signature validation is required; raw body must not be parsed
```

### 7.4 Handoff Packets

When one agent completes a subtask that another will continue:

```json
// .agent/handoffs/feat-auth-mfa-codex→claude.json
{
  "from": "codex",
  "to": "claude-code",
  "task": "feat-auth-mfa",
  "subtask": "st-001→st-002",
  "timestamp": "2025-04-19T10:22:00Z",
  "summary": "Scaffolded MFA migration. Added mfa_factors table with TOTP secret column (encrypted). Migration is reversible. No RLS policies yet — that's st-002's job.",
  "artifacts": [
    "supabase/migrations/20250418_mfa.sql"
  ],
  "warnings": [
    "TOTP secret column uses pgcrypto — ensure extension is enabled in prod"
  ],
  "next_agent_context": "Review migration, implement verify_totp(user_id, code) in src/lib/auth/mfa.ts using the otplib package (already in package.json). RLS policy for mfa_factors should restrict SELECT to auth.uid() = user_id."
}
```

---

## 8. CLI Automation Scripts

### 8.1 `wf` — Worktree Flow CLI

Install this as a shell function or standalone script at `~/bin/wf`:

```bash
#!/usr/bin/env bash
# ~/bin/wf — Worktree Flow CLI
# Usage: wf <command> [args]

set -euo pipefail

BARE_REPO="$HOME/projects/myapp.git"
WT_BASE="$HOME/projects/myapp/wt"
AGENT_DIR="$HOME/projects/myapp/.agent"
CONFIG_DIR="$HOME/.config/myapp-agents"

cmd="${1:-help}"
shift || true

case "$cmd" in

  # ── Create a new worktree ──────────────────────────────────────────────
  new)
    TYPE="$1"        # feat|fix|exp|audit|refactor|chore
    SCOPE="$2"       # e.g. auth-mfa
    BASE="${3:-develop}"
    SLUG="$TYPE-$SCOPE"
    BRANCH="$TYPE/$SCOPE"
    WT_PATH="$WT_BASE/$SLUG"

    echo "→ Creating worktree: $SLUG (branch: $BRANCH)"
    git -C "$BARE_REPO" worktree add -b "$BRANCH" "$WT_PATH" "$BASE"

    # Copy shared git hooks
    cp "$CONFIG_DIR/hooks/"* "$WT_PATH/.git/hooks/" 2>/dev/null || true

    # Install dependencies (offline-first)
    (cd "$WT_PATH" && pnpm install --prefer-offline --silent)

    # Create task file skeleton
    cat > "$AGENT_DIR/tasks/$SLUG.yaml" <<YAML
task:
  id: $SLUG
  title: ""
  type: ${TYPE}
  worktree: wt/$SLUG
  branch: $BRANCH
  base_branch: $BASE
  status: created
  assigned_agent: unassigned
YAML

    # Update state.json
    node -e "
      const fs = require('fs');
      const s = JSON.parse(fs.readFileSync('$AGENT_DIR/state.json', 'utf8'));
      s.worktrees['$SLUG'] = {
        branch: '$BRANCH',
        status: 'created',
        assigned_agent: 'unassigned',
        task_file: '.agent/tasks/$SLUG.yaml',
        human_gate: 'none',
        created_at: new Date().toISOString()
      };
      fs.writeFileSync('$AGENT_DIR/state.json', JSON.stringify(s, null, 2));
    "

    echo "✓ Worktree ready at $WT_PATH"
    echo "  Run: wf assign $SLUG <agent>  to assign an agent"
    echo "  Run: wf session $SLUG         to open a Zellij session"
    ;;

  # ── Assign an agent to a worktree ─────────────────────────────────────
  assign)
    SLUG="$1"
    AGENT="$2"  # hermes|claude-code|codex|opencode
    WT_PATH="$WT_BASE/$SLUG"

    echo "→ Assigning $AGENT to $SLUG"

    # Write agent config into the worktree
    echo "AGENT=$AGENT" > "$WT_PATH/.agent-config"
    echo "TASK_FILE=$AGENT_DIR/tasks/$SLUG.yaml" >> "$WT_PATH/.agent-config"
    echo "LOG_FILE=$AGENT_DIR/logs/$AGENT.log" >> "$WT_PATH/.agent-config"

    # Update state.json
    node -e "
      const fs = require('fs');
      const s = JSON.parse(fs.readFileSync('$AGENT_DIR/state.json', 'utf8'));
      s.worktrees['$SLUG'].assigned_agent = '$AGENT';
      s.worktrees['$SLUG'].status = 'active';
      fs.writeFileSync('$AGENT_DIR/state.json', JSON.stringify(s, null, 2));
    "

    echo "✓ $AGENT assigned to $SLUG"
    ;;

  # ── Launch a Zellij session for a worktree ────────────────────────────
  session)
    SLUG="$1"
    WT_PATH="$WT_BASE/$SLUG"
    SESSION="wt-$SLUG"

    if zellij list-sessions 2>/dev/null | grep -q "$SESSION"; then
      echo "→ Attaching to existing session: $SESSION"
      zellij attach "$SESSION"
    else
      echo "→ Creating new Zellij session: $SESSION"
      zellij --session "$SESSION" \
        --layout "$CONFIG_DIR/layouts/worktree.kdl" \
        options --default-cwd "$WT_PATH"
    fi
    ;;

  # ── Rebase a worktree branch from its base ────────────────────────────
  sync)
    SLUG="$1"
    WT_PATH="$WT_BASE/$SLUG"

    BASE=$(grep 'base_branch:' "$AGENT_DIR/tasks/$SLUG.yaml" | awk '{print $2}')
    echo "→ Syncing $SLUG from $BASE"

    git -C "$BARE_REPO" fetch origin

    (
      cd "$WT_PATH"
      git fetch origin
      # Stash any agent-in-progress changes
      git stash push -m "wf-sync-stash-$(date +%s)" || true
      git rebase "origin/$BASE"
      git stash pop || true
    )

    echo "✓ $SLUG synced and rebased on $BASE"
    ;;

  # ── Freeze a worktree (block agent writes pending human review) ────────
  freeze)
    SLUG="$1"
    REASON="${2:-manual freeze}"
    WT_PATH="$WT_BASE/$SLUG"

    # Make worktree read-only for the agent user
    chmod -R a-w "$WT_PATH/src" 2>/dev/null || true
    touch "$WT_PATH/.agent-frozen"
    echo "$REASON" > "$WT_PATH/.agent-frozen"

    node -e "
      const fs = require('fs');
      const s = JSON.parse(fs.readFileSync('$AGENT_DIR/state.json', 'utf8'));
      s.worktrees['$SLUG'].status = 'frozen';
      s.worktrees['$SLUG'].freeze_reason = '$REASON';
      fs.writeFileSync('$AGENT_DIR/state.json', JSON.stringify(s, null, 2));
    "

    echo "❄  $SLUG frozen: $REASON"
    ;;

  # ── Unfreeze a worktree ───────────────────────────────────────────────
  unfreeze)
    SLUG="$1"
    WT_PATH="$WT_BASE/$SLUG"

    chmod -R u+w "$WT_PATH/src" 2>/dev/null || true
    rm -f "$WT_PATH/.agent-frozen"

    node -e "
      const fs = require('fs');
      const s = JSON.parse(fs.readFileSync('$AGENT_DIR/state.json', 'utf8'));
      s.worktrees['$SLUG'].status = 'active';
      delete s.worktrees['$SLUG'].freeze_reason;
      fs.writeFileSync('$AGENT_DIR/state.json', JSON.stringify(s, null, 2));
    "

    echo "✓ $SLUG unfrozen"
    ;;

  # ── Destroy a completed/abandoned worktree ────────────────────────────
  destroy)
    SLUG="$1"
    WT_PATH="$WT_BASE/$SLUG"
    BRANCH=$(grep 'branch:' "$AGENT_DIR/tasks/$SLUG.yaml" | head -1 | awk '{print $2}')

    read -rp "Destroy $SLUG (branch: $BRANCH)? [y/N] " confirm
    [[ "$confirm" != "y" ]] && echo "Aborted." && exit 0

    git -C "$BARE_REPO" worktree remove "$WT_PATH" --force
    git -C "$BARE_REPO" worktree prune

    # Archive task file instead of deleting
    mkdir -p "$AGENT_DIR/tasks/archived"
    mv "$AGENT_DIR/tasks/$SLUG.yaml" "$AGENT_DIR/tasks/archived/"

    node -e "
      const fs = require('fs');
      const s = JSON.parse(fs.readFileSync('$AGENT_DIR/state.json', 'utf8'));
      delete s.worktrees['$SLUG'];
      fs.writeFileSync('$AGENT_DIR/state.json', JSON.stringify(s, null, 2));
    "

    echo "✓ Worktree $SLUG destroyed and archived"
    ;;

  # ── Status overview ───────────────────────────────────────────────────
  status)
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                  WORKTREE FLOW STATUS                       ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    git -C "$BARE_REPO" worktree list
    echo ""
    node -e "
      const s = JSON.parse(require('fs').readFileSync('$AGENT_DIR/state.json', 'utf8'));
      const wts = s.worktrees;
      const statusIcons = {
        active: '🟢', created: '🔵', frozen: '❄️ ',
        review: '🟡', merged: '✅', abandoned: '⚫'
      };
      for (const [slug, wt] of Object.entries(wts)) {
        const icon = statusIcons[wt.status] || '❓';
        console.log(\`\${icon} \${slug.padEnd(30)} agent: \${(wt.assigned_agent||'none').padEnd(15)} gate: \${wt.human_gate}\`);
      }
      console.log('');
      console.log('API Health:');
      for (const [agent, health] of Object.entries(s.api_health||{})) {
        const hIcon = health === 'healthy' ? '✅' : health === 'degraded' ? '⚠️' : '❌';
        console.log(\`  \${hIcon} \${agent}: \${health}\`);
      }
    "
    ;;

  help|*)
    echo ""
    echo "wf — Worktree Flow CLI"
    echo ""
    echo "Commands:"
    echo "  wf new <type> <scope> [base]  Create a new worktree"
    echo "  wf assign <slug> <agent>      Assign agent to worktree"
    echo "  wf session <slug>             Open Zellij session"
    echo "  wf sync <slug>                Rebase from base branch"
    echo "  wf freeze <slug> [reason]     Freeze worktree (block agent)"
    echo "  wf unfreeze <slug>            Unfreeze worktree"
    echo "  wf destroy <slug>             Remove worktree + archive task"
    echo "  wf status                     Show all worktrees + API health"
    echo ""
    ;;
esac
```

```bash
chmod +x ~/bin/wf
# Add ~/bin to PATH in your shell config if not already there
```

### 8.2 Agent Invocation Wrappers

```bash
#!/usr/bin/env bash
# ~/bin/agent-run — Invoke the correct agent for a worktree's current subtask

SLUG="$1"
AGENT_DIR="$HOME/projects/myapp/.agent"
WT_PATH="$HOME/projects/myapp/wt/$SLUG"

source "$WT_PATH/.agent-config"

# Check for freeze
if [[ -f "$WT_PATH/.agent-frozen" ]]; then
  echo "❌ Worktree $SLUG is frozen: $(cat "$WT_PATH/.agent-frozen")"
  exit 1
fi

# Check API health before invoking
health=$(node -e "
  const s = JSON.parse(require('fs').readFileSync('$AGENT_DIR/state.json', 'utf8'));
  console.log(s.api_health['$AGENT'] || 'unknown');
")

if [[ "$health" == "degraded" || "$health" == "down" ]]; then
  echo "⚠️  $AGENT API is $health — checking fallback..."
  # Read fallback from routing.yaml and reassign
  AGENT=$(yq ".fallback_chain.$AGENT[0]" ~/.config/myapp-agents/routing.yaml)
  echo "→ Falling back to $AGENT"
fi

case "$AGENT" in
  claude-code)
    cd "$WT_PATH"
    claude \
      --context-file "$AGENT_DIR/memory/codebase.md" \
      --context-file "$AGENT_DIR/memory/conventions.md" \
      --task-file "$TASK_FILE" \
      2>&1 | tee -a "$LOG_FILE"
    ;;

  codex)
    cd "$WT_PATH"
    TASK_PROMPT=$(yq ".plan.subtasks[] | select(.status == \"pending\") | .title" "$TASK_FILE" | head -1)
    openai codex \
      --model gpt-5.3-codex \
      --prompt "$TASK_PROMPT" \
      --context "$(cat $AGENT_DIR/memory/codebase.md)" \
      2>&1 | tee -a "$LOG_FILE"
    ;;

  opencode)
    cd "$WT_PATH"
    opencode run \
      --model auto \
      --task-file "$TASK_FILE" \
      2>&1 | tee -a "$LOG_FILE"
    ;;
esac
```

### 8.3 Sync-All Script

```bash
#!/usr/bin/env bash
# ~/bin/sync-all-wts — Rebase all active worktrees from their base branches

AGENT_DIR="$HOME/projects/myapp/.agent"

node -e "
  const s = JSON.parse(require('fs').readFileSync('$AGENT_DIR/state.json', 'utf8'));
  Object.entries(s.worktrees)
    .filter(([_, wt]) => wt.status === 'active')
    .map(([slug]) => slug)
    .forEach(slug => {
      const { execSync } = require('child_process');
      try {
        console.log('Syncing:', slug);
        execSync('wf sync ' + slug, { stdio: 'inherit' });
      } catch (e) {
        console.error('⚠️  Sync failed for', slug, '— skipping');
      }
    });
"
```

---

## 9. Human-in-the-Loop Checkpoints

### 9.1 Gate Types

| Gate Type | Trigger | Blocks | Required Approval |
|---|---|---|---|
| `security_review` | Any `touches_auth`, `touches_phi`, or `security_critical` flag | Merge to develop | Human + Claude Code review |
| `pre_merge` | Any PR from feature/fix → develop | Merge | Human only |
| `pre_release` | Any PR from develop → main | Merge | Human + CI green |
| `conflict_resolution` | Agent arbitration failure | Agent execution | Human |
| `scope_change` | Agent detects task expanded > 20% | Agent execution | Human re-plan via Hermes |
| `cost_threshold` | >100 API calls/day on one worktree | Agent execution | Human cost approval |

### 9.2 Security Review Checklist (Pre-Merge Gate)

```markdown
## Security Review — feat/auth-mfa

**Reviewer:** [human]
**Agent Reviewer:** claude-code
**Date:** ___________

### Auth & Session
- [ ] JWT claims validated server-side (not just client)
- [ ] Session invalidation on MFA setup/removal
- [ ] TOTP secret stored encrypted at rest (pgcrypto)
- [ ] Backup codes hashed with bcrypt (not TOTP secret format)
- [ ] Rate limiting on /api/auth/mfa/verify (max 5 attempts/10min)

### Database
- [ ] RLS policies reviewed: SELECT/INSERT/UPDATE/DELETE on mfa_factors
- [ ] No service role key usage outside edge functions
- [ ] Migration is reversible (has down migration)

### API Layer
- [ ] Webhook signatures verified (raw body, not parsed)
- [ ] All user inputs validated with Zod before DB write
- [ ] Error messages don't leak internal state (e.g. "user not found" → generic)

### Code Quality
- [ ] No hardcoded secrets or API keys in diff
- [ ] TypeScript strict mode: no `any` introduced
- [ ] All new code paths have tests (unit + integration)

**Decision:** [ ] Approve  [ ] Request Changes  [ ] Escalate

**Notes:**
___________________________________________
```

### 9.3 Approval Gate Script

```bash
#!/usr/bin/env bash
# Called by pre-push hook or manually: gate-check <slug>

SLUG="$1"
AGENT_DIR="$HOME/projects/myapp/.agent"
TASK_FILE="$AGENT_DIR/tasks/$SLUG.yaml"

GATE_REQUIRED=$(yq '.human_gates[] | select(.status == "pending") | .type' "$TASK_FILE")

if [[ -n "$GATE_REQUIRED" ]]; then
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  HUMAN GATE REQUIRED: $GATE_REQUIRED"
  echo "  Worktree: $SLUG"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
  echo "Review the changes:"
  echo "  cd ~/projects/myapp/wt/$SLUG && git diff develop..HEAD"
  echo ""
  read -rp "Approve this gate? [y/N] " confirm

  if [[ "$confirm" == "y" ]]; then
    yq -i ".human_gates[] |= select(.status == \"pending\") .status = \"approved\"" "$TASK_FILE"
    echo "✅ Gate approved. Push unblocked."
  else
    echo "❌ Gate rejected. Worktree frozen for revision."
    wf freeze "$SLUG" "gate-$GATE_REQUIRED rejected by human"
    exit 1
  fi
fi
```

### 9.4 Pre-Push Hook

```bash
#!/usr/bin/env bash
# .git/hooks/pre-push (copied to each worktree via wf new)

SLUG=$(basename "$(pwd)")
"$HOME/bin/gate-check" "$SLUG"
```

---

## 10. Code Quality & Security Workflows

### 10.1 Per-Worktree Quality Pipeline

Every worktree runs this pipeline before a human gate is presented:

```bash
#!/usr/bin/env bash
# ~/bin/wf-qa <slug> — Run full QA pipeline on a worktree

SLUG="$1"
WT_PATH="$HOME/projects/myapp/wt/$SLUG"
PASS=true

echo "━━━ QA Pipeline: $SLUG ━━━"

cd "$WT_PATH"

# 1. TypeScript type check
echo "→ TypeScript..."
pnpm tsc --noEmit 2>&1 | tail -5 || PASS=false

# 2. ESLint
echo "→ ESLint..."
pnpm eslint src --max-warnings=0 2>&1 | tail -10 || PASS=false

# 3. Unit tests
echo "→ Unit tests..."
pnpm vitest run --reporter=verbose 2>&1 | tail -20 || PASS=false

# 4. Zod schema validation (custom check for broken schemas)
echo "→ Zod schemas..."
pnpm ts-node scripts/validate-schemas.ts 2>&1 || PASS=false

# 5. Dependency audit
echo "→ pnpm audit..."
pnpm audit --audit-level=high 2>&1 | tail -10 || PASS=false

# 6. Secrets scan
echo "→ Secrets scan (gitleaks)..."
gitleaks detect --source . --no-git --redact 2>&1 | tail -5 || PASS=false

# 7. Static analysis (optional, for security-flagged worktrees)
FLAGS=$(yq '.task.flags[]' "$HOME/projects/myapp/.agent/tasks/$SLUG.yaml" 2>/dev/null)
if echo "$FLAGS" | grep -q "security_critical"; then
  echo "→ Semgrep (security rules)..."
  semgrep --config=p/nextjs --config=p/typescript \
    --no-autofix --json src/ 2>&1 | \
    node -e "
      const r = JSON.parse(require('fs').readFileSync('/dev/stdin','utf8'));
      if (r.results?.length > 0) {
        console.error('Semgrep findings:', r.results.length);
        process.exit(1);
      }
    " || PASS=false
fi

# 8. Bundle size check (warn only)
echo "→ Bundle size..."
pnpm build 2>&1 | grep "Route\|Size\|First" | tail -20

echo ""
if [[ "$PASS" == "true" ]]; then
  echo "✅ QA Pipeline PASSED for $SLUG"
else
  echo "❌ QA Pipeline FAILED for $SLUG — review output above"
  exit 1
fi
```

### 10.2 Recommended Tool Stack

```bash
# Install once globally
pnpm add -g gitleaks        # Secrets scanning
pip install semgrep          # Static analysis
pnpm add -g knip             # Dead code / unused exports detector

# Project-level (in package.json)
{
  "devDependencies": {
    "vitest": "^2.x",
    "@biomejs/biome": "^1.x",   # Faster ESLint + Prettier replacement
    "knip": "^5.x",
    "zod": "^3.x"
  }
}
```

### 10.3 Biome Configuration (Replaces ESLint + Prettier)

```json
// biome.json (monorepo root, shared across all worktrees)
{
  "$schema": "https://biomejs.dev/schemas/1.8.0/schema.json",
  "organizeImports": { "enabled": true },
  "linter": {
    "enabled": true,
    "rules": {
      "recommended": true,
      "security": { "noEval": "error" },
      "suspicious": { "noExplicitAny": "warn" }
    }
  },
  "formatter": {
    "enabled": true,
    "indentStyle": "space",
    "indentWidth": 2,
    "lineWidth": 100
  }
}
```

---

## 11. Zellij Session Architecture

### 11.1 Worktree Layout (`worktree.kdl`)

```kdl
// ~/.config/myapp-agents/layouts/worktree.kdl

layout {
  pane split_direction="vertical" {

    // Left column: editor (60%)
    pane size="60%" {
      command "nvim"
      args "."
    }

    // Right column: split into 3 (40%)
    pane split_direction="horizontal" size="40%" {

      // Top-right: shell / agent output
      pane size="50%" {
        name "agent"
      }

      // Middle-right: git status / diff
      pane size="25%" {
        command "lazygit"
      }

      // Bottom-right: QA / logs
      pane size="25%" {
        name "qa"
        command "tail"
        args "-f" {$AGENT_DIR}/logs/claude-code.log
      }
    }
  }

  // Bottom bar: persistent status pane
  pane size=2 borderless=true {
    plugin location="zellij:status-bar"
  }
}
```

### 11.2 Multi-Worktree Overview Session

```kdl
// ~/.config/myapp-agents/layouts/overview.kdl
// Launch with: zellij --session overview --layout overview.kdl

layout {
  pane split_direction="horizontal" {
    pane {
      command "watch"
      args "-n2" "wf status"
      name "status"
    }
    pane {
      command "tail"
      args "-f"
           {$AGENT_DIR}/logs/hermes.log
           {$AGENT_DIR}/logs/claude-code.log
      name "logs"
    }
  }
}
```

### 11.3 Session Management Cheatsheet

```bash
# Open a worktree session
wf session feat-auth-mfa

# List all sessions
zellij list-sessions

# Detach from current session (keeps it alive)
<Ctrl-O> d

# Attach to a specific session
zellij attach wt-feat-auth-mfa

# Kill a session after worktree is destroyed
zellij kill-session wt-feat-auth-mfa

# Open the overview dashboard
zellij --session overview --layout ~/.config/myapp-agents/layouts/overview.kdl
```

---

## 12. Failure Handling Strategies

### 12.1 Agent API Degradation

```bash
# ~/bin/health-check — Run on cron (*/5 * * * *) or before agent invocations

check_agent() {
  local agent="$1"
  local endpoint="$2"
  local timeout=5

  if curl -sf --max-time "$timeout" "$endpoint" > /dev/null 2>&1; then
    echo "healthy"
  else
    echo "degraded"
  fi
}

update_health() {
  local agent="$1"
  local status="$2"
  node -e "
    const fs = require('fs');
    const s = JSON.parse(fs.readFileSync('$AGENT_DIR/state.json', 'utf8'));
    s.api_health = s.api_health || {};
    s.api_health['$agent'] = '$status';
    fs.writeFileSync('$AGENT_DIR/state.json', JSON.stringify(s, null, 2));
  "
}

HERMES_STATUS=$(check_agent hermes "https://api.hermes.ai/health")
CLAUDE_STATUS=$(check_agent claude-code "https://api.anthropic.com/v1/health")
CODEX_STATUS=$(check_agent codex "https://api.openai.com/v1/models")
OC_STATUS=$(check_agent opencode "https://openrouter.ai/api/v1/health")

update_health "hermes" "$HERMES_STATUS"
update_health "claude-code" "$CLAUDE_STATUS"
update_health "codex" "$CODEX_STATUS"
update_health "opencode" "$OC_STATUS"
```

### 12.2 Merge Conflict Resolution Protocol

When a rebase produces conflicts:

```bash
# wf sync detects conflicts and enters this protocol

1. AUTO-RESOLVE: Run git rerere (reuse recorded resolutions)
   git config rerere.enabled true  # Set once in bare repo

2. AGENT-ASSIST: If rerere fails, pipe conflict markers to Claude Code
   claude --prompt "Resolve this merge conflict. Context: $(cat MERGE_MSG)" \
          --input "$(cat conflicted-file.ts)"

3. HUMAN ESCALATE: If agent confidence < threshold or security-flagged file
   wf freeze <slug> "merge-conflict: needs human resolution"
   # Notify via desktop notification
   notify-send "Merge Conflict" "Manual resolution needed: $SLUG"
```

### 12.3 Agent Conflict Arbitration

```
Conflict detected (two agents wrote to same file in different subtasks)
         │
         ▼
1. Both worktrees frozen automatically
2. Hermes invoked as arbiter:
   - Reads both diffs
   - Reads original task spec
   - Produces resolution commit
3. If Hermes confidence < 0.8:
   - Human escalation triggered
   - Both diffs presented side-by-side in terminal
   - Human picks winner or edits manually
4. After resolution:
   - Winning changes merged into canonical worktree
   - Losing worktree rebased from canonical
   - Both unfrozen
```

### 12.4 Runaway Agent Circuit Breaker

```bash
# Pre-commit hook: detect suspiciously large agent commits

DIFF_LINES=$(git diff --cached --stat | tail -1 | grep -oP '\d+ insertion' | grep -oP '\d+')

if [[ ${DIFF_LINES:-0} -gt 500 ]]; then
  echo "⚠️  Large commit detected ($DIFF_LINES lines). Triggering human review."
  SLUG=$(basename "$(pwd)")
  wf freeze "$SLUG" "circuit-breaker: large diff ($DIFF_LINES lines)"
  notify-send "Circuit Breaker" "$SLUG: Agent produced $DIFF_LINES line diff — review required"
  exit 1
fi
```

---

## 13. Performance & Resource Optimization

### 13.1 Laptop-Specific Tuning

```bash
# ~/.config/myapp-agents/resource-limits.sh
# Source this before starting agent sessions

# Cap Node.js memory per worktree process
export NODE_OPTIONS="--max-old-space-size=1024"

# pnpm: use shared store, avoid duplicate installs
export PNPM_HOME="$HOME/.pnpm-store"

# Vitest: run only changed tests (watch mode off by default)
export VITEST_CHANGED_ONLY=true

# Turbo: limit parallel build jobs to half of CPU cores
CORES=$(nproc)
export TURBO_CONCURRENCY=$((CORES / 2))

# Limit concurrent agent sessions (max 3 active worktrees at once)
ACTIVE_COUNT=$(node -e "
  const s = JSON.parse(require('fs').readFileSync('$AGENT_DIR/state.json', 'utf8'));
  console.log(Object.values(s.worktrees).filter(w => w.status === 'active').length);
")

if [[ "$ACTIVE_COUNT" -ge 3 ]]; then
  echo "⚠️  3 active worktrees already running. Consider pausing one before creating new sessions."
fi
```

### 13.2 API Cost Controls

```yaml
# ~/.config/myapp-agents/cost-limits.yaml

daily_limits:
  hermes: 50          # calls/day (expensive model)
  claude-code: 150    # calls/day
  codex: 500          # calls/day (cheaper)
  opencode: 1000      # calls/day (cheapest via OpenRouter)

per_worktree_daily: 100   # Hard cap per worktree
monthly_budget_usd: 200   # Soft alert threshold

alerts:
  on_80_percent: notify    # Desktop notification
  on_100_percent: freeze   # Freeze all worktrees, require human override
```

### 13.3 Context Window Efficiency

- **Never pass full file contents to Codex** — use Hermes to extract the relevant 50-100 lines first
- **Claude Code context priority:** task file → handoff packet → specific source files → memory digest
- **Memory files are summarized nightly** by OpenCode to prevent bloat:

```bash
# ~/bin/nightly-memory-compress
# Add to cron: 0 2 * * * ~/bin/nightly-memory-compress

cd "$HOME/projects/myapp/.agent/memory"
for file in *.md; do
  LINES=$(wc -l < "$file")
  if [[ "$LINES" -gt 500 ]]; then
    echo "Compressing $file ($LINES lines)..."
    opencode summarize \
      --input "$file" \
      --max-lines 300 \
      --preserve-structure \
      --output "$file.compressed"
    mv "$file" "$file.$(date +%Y%m%d).bak"
    mv "$file.compressed" "$file"
  fi
done
```

### 13.4 Git Performance

```bash
# Run in bare repo after many worktrees cycle through
git -C ~/projects/myapp.git gc --aggressive --prune=now

# Enable commit graph for faster log/blame operations
git -C ~/projects/myapp.git config core.commitGraph true
git -C ~/projects/myapp.git commit-graph write --reachable --changed-paths

# Partial clone (if repo is very large) — fetch blobs on demand
git clone --filter=blob:none --bare git@github.com:yourorg/myapp.git ~/projects/myapp.git
```

---

## 14. Quick Reference Card

```
WORKTREE LIFECYCLE
─────────────────────────────────────────────────────────
wf new feat auth-mfa          Create feat/auth-mfa worktree
wf assign feat-auth-mfa claude-code   Assign agent
wf session feat-auth-mfa      Open Zellij session
agent-run feat-auth-mfa       Run assigned agent
wf sync feat-auth-mfa         Rebase from develop
wf-qa feat-auth-mfa           Run full QA pipeline
gate-check feat-auth-mfa      Trigger human approval gate
wf destroy feat-auth-mfa      Remove + archive after merge

AGENT ROUTING (quick reference)
─────────────────────────────────────────────────────────
Hermes       → Planning, decomposition, arbitration
Claude Code  → Auth, RLS, refactoring, security review
Codex        → Components, migrations, tests, scaffolding
OpenCode     → Docs, experiments, overflow

GATE TRIGGERS
─────────────────────────────────────────────────────────
touches_auth / touches_phi / security_critical → claude-code review + human gate
> 500 line diff                                → circuit breaker + human review
merge conflict after rerere + agent fail       → human escalation
subtask count grows > 20% of original plan     → hermes re-plan + human confirm

DAILY MAINTENANCE
─────────────────────────────────────────────────────────
sync-all-wts                  Rebase all active worktrees
~/bin/health-check            Verify API health
~/bin/nightly-memory-compress Compress memory files (cron)
git -C ~/projects/myapp.git gc --prune=now   (weekly)
```

---

*Designed for Lintra LLC engineering — Next.js / TypeScript / Supabase / Clerk monorepo*
*Last updated: April 2025 — schema v1.2*