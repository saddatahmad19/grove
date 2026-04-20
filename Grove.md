# Grove — Multi-Agent Worktree TUI

## Complete Architecture, Design Decisions & Code Scaffold

> **Stack:** Go + Bubble Tea + Lip Gloss + creack/pty  
> **Builds on:** openharness patterns (PTY + leader key)  
> **Purpose:** Manage git worktrees and multiple AI agent sessions from a single terminal interface

---

## Table of Contents

1. [Why Go + Bubble Tea](#1-why-go--bubble-tea)
2. [Mental Model & UI Concept](#2-mental-model--ui-concept)
3. [Project Structure](#3-project-structure)
4. [Module Dependency Map](#4-module-dependency-map)
5. [State Architecture](#5-state-architecture)
6. [UI Layout & Navigation Model](#6-ui-layout--navigation-model)
7. [Core Package: `worktree/`](#7-core-package-worktree)
8. [Core Package: `agent/`](#8-core-package-agent)
9. [Core Package: `session/`](#9-core-package-session)
10. [Core Package: `git/`](#10-core-package-git)
11. [UI Package: `ui/model.go`](#11-ui-package-uimodelgo)
12. [UI Package: `ui/update.go`](#12-ui-package-uiupdatego)
13. [UI Package: `ui/view.go`](#13-ui-package-uiviewgo)
14. [UI Package: `ui/styles.go`](#14-ui-package-uistylesgo)
15. [Config System](#15-config-system)
16. [main.go Entrypoint](#16-maingo-entrypoint)
17. [Key Pitfalls & Critical Patterns](#17-key-pitfalls--critical-patterns)
18. [Build Phases](#18-build-phases)
19. [Keybinding Reference](#19-keybinding-reference)

---

## 1. Why Go + Bubble Tea

Given your existing `openharness` codebase, Go + Bubble Tea is the **only correct answer** here. The reasons are concrete:

**You already have the hard parts solved.** The PTY + Bubble Tea input loop conflict, the leader key state machine, the `waitForOutput` goroutine pattern, window resize propagation — all of these are solved in openharness. Grove is openharness with a two-panel layout and hierarchical state (worktrees → agents) instead of a flat tab list.

**Go's goroutine model is perfect for this.** Each agent PTY session needs its own read goroutine. Go channels map cleanly onto Bubble Tea's `tea.Cmd` message-passing model. You get real concurrency without the async/callback hell you'd hit in a Rust or Node.js equivalent.

**Alternatives and why they lose:**

|Alternative|Why it loses|
|---|---|
|Rust + Ratatui|Would require rewriting PTY patterns from scratch; no openharness to build on|
|Python + Textual|Too slow for real-time PTY output; GIL causes latency spikes|
|Node.js + Ink|No real PTY support; event loop bottlenecks under multiple simultaneous agents|
|Shell + fzf|Cannot manage stateful PTY sessions; no proper TUI layout|
|Zellij plugin (Rust WASM)|WASM sandbox prevents direct process spawning|

---

## 2. Mental Model & UI Concept

Grove is a **two-panel, mode-driven terminal multiplexer** specialized for the worktree + agent workflow.

```
┌─────────────────────────────────────────────────────────────────────┐
│ grove                                          feat-auth-mfa  [●]   │
├──────────────────┬──────────────────────────────────────────────────┤
│ WORKTREES    [3] │ [claude-code ●] [codex] [hermes]                 │
│                  ├──────────────────────────────────────────────────┤
│ ▶ feat-auth-mfa  │                                                   │
│   fix-rls-policy │  $ claude --task-file .agent/tasks/feat-auth-mfa │
│   exp-edge-rt    │  ✓ Reading codebase context...                   │
│                  │  ✓ Analyzing src/lib/auth/session.ts             │
│                  │  → Implementing TOTP verification...             │
│ ── METADATA ──   │                                                   │
│ Branch:          │  function verifyTOTP(secret: string,             │
│  feat/auth-mfa   │    code: string): boolean {                      │
│ Agent: claude    │    // Using otplib with 30s window               │
│ Gate: pending    │    return authenticator.verify({ token: code,    │
│ Status: active   │      secret, encoding: 'base32' })               │
│                  │  }                                                │
│                  │                                                   │
│                  │  ▊                                                │
├──────────────────┴──────────────────────────────────────────────────┤
│ [N]av [T]erminal  │  Ctrl+B: cmd  wt:feat-auth-mfa  claude-code  ✓  │
└─────────────────────────────────────────────────────────────────────┘
```

### The Three Modes

```
         ┌──────────────────────────────────────────────────────┐
         │                    NAVIGATE MODE                      │
         │  - Arrow keys move worktree selection                 │
         │  - Tab cycles agent tabs                              │
         │  - Enter enters Terminal mode for active session      │
         │  - All Ctrl+B commands available                      │
         └──────────────┬──────────────────────┬────────────────┘
                        │                      │
              Enter / Ctrl+T           Ctrl+B → command
                        │                      │
         ┌──────────────▼──────┐    ┌──────────▼──────────────┐
         │   TERMINAL MODE     │    │    COMMAND MODE          │
         │                     │    │                          │
         │  All keys forwarded │    │  Text input for:         │
         │  directly to PTY    │    │  - New worktree name     │
         │                     │    │  - Agent assignment      │
         │  Escape → Navigate  │    │  - Sync/destroy confirm  │
         │                     │    │                          │
         └─────────────────────┘    └──────────────────────────┘
```

**The key insight:** In Navigate mode you control Grove. In Terminal mode you control the agent. `Escape` always brings you back to Navigate mode without killing the PTY — the agent keeps running.

---

## 3. Project Structure

```
grove/
├── main.go                        # Entrypoint
├── go.mod
├── go.sum
│
├── config/
│   └── config.go                  # JSONC config + Agent definitions
│
├── git/
│   └── git.go                     # Thin wrapper around git CLI (worktree ops)
│
├── worktree/
│   └── worktree.go                # Worktree struct + state loader
│
├── agent/
│   └── agent.go                   # Agent profiles + command builder
│
├── session/
│   └── session.go                 # PTY session (from openharness, extended)
│
├── state/
│   └── state.go                   # Read/write .agent/state.json
│
└── ui/
    ├── model.go                   # Model struct — the full app state
    ├── update.go                  # Update() — the state machine
    ├── view.go                    # View() — top-level rendering
    ├── keys.go                    # All key constants
    ├── styles.go                  # All lipgloss styles
    ├── messages.go                # All tea.Msg type definitions
    └── panels/
        ├── sidebar.go             # Worktree list panel renderer
        ├── agenttabs.go           # Agent tab bar renderer
        ├── terminal.go            # PTY output renderer
        ├── statusbar.go           # Bottom status bar renderer
        ├── modal.go               # Command/leader modal
        └── metadata.go            # Worktree metadata panel (in sidebar)
```

---

## 4. Module Dependency Map

```
main.go
  └── config.Load()              → reads ~/.config/grove/config.jsonc
  └── state.Load()               → reads .agent/state.json
  └── worktree.LoadAll()         → git worktree list + state merge
  └── ui.New(cfg, wts, state)    → builds initial model
  └── tea.NewProgram(model)      → starts event loop

ui/model.go
  └── []Worktree                 → from worktree package
      └── []AgentSession         → from session package
          └── *os.File (PTY)     → from creack/pty
  └── *config.Config             → agent defs, bare repo path
  └── *state.State               → synced to disk on mutation

worktree/worktree.go
  └── git.ListWorktrees()        → parses `git worktree list --porcelain`
  └── git.AddWorktree()          → `git worktree add -b <branch> <path> <base>`
  └── git.RemoveWorktree()       → `git worktree remove --force`

session/session.go
  └── agent.BuildCommand()       → constructs the CLI invocation
  └── pty.Start()                → spawns the PTY
  └── waitForOutput()            → goroutine → tea.Cmd pipeline
```

```go
// go.mod
module github.com/yourusername/grove

go 1.23

require (
    github.com/charmbracelet/bubbletea  v0.27.0
    github.com/charmbracelet/bubbles    v0.20.0
    github.com/charmbracelet/lipgloss   v1.0.0
    github.com/creack/pty               v1.1.21
    tailscale.com/util/hujson           v0.0.0-20240710205705-3d955e7c9e04
    github.com/google/uuid              v1.6.0
)
```

---

## 5. State Architecture

This is the most important section. Grove has **two layers of state**: the in-memory model (what Bubble Tea owns) and the on-disk state (`.agent/state.json` from the workflow doc). They must stay in sync.

### 5.1 The Worktree State Hierarchy

```
Model
└── worktrees []Worktree          ← slice ordered by creation time
    ├── Path string                ← absolute FS path
    ├── Branch string              ← e.g. "feat/auth-mfa"
    ├── Slug string                ← e.g. "feat-auth-mfa"
    ├── Status WorktreeStatus      ← active|frozen|review|merged
    ├── HumanGate GateStatus       ← none|pending|required|approved
    ├── Flags []string             ← touches_auth, security_critical, etc.
    ├── TaskFile string            ← path to .agent/tasks/<slug>.yaml
    └── sessions []*AgentSession   ← PTY sessions open in this worktree
        ├── ID string
        ├── AgentName string       ← "claude-code"|"codex"|"hermes"|"opencode"
        ├── PTY *os.File
        ├── Output []byte          ← scrollback buffer
        └── Done bool
```

### 5.2 The Navigation State

```
Model
├── activeWorktreeIdx int          ← which worktree is selected in sidebar
├── activeSessionIdx int           ← which agent tab is active (per worktree)
├── mode AppMode                   ← Navigate | Terminal | Command
├── leaderState LeaderState        ← None | Active | AwaitingInput
└── cmdInput textinput.Model       ← for Command mode text entry
```

### 5.3 State Transitions

```
                   ┌─────────────────────┐
                   │   NAVIGATE MODE     │
                   │                     │
     ┌─────────────┤  ↑↓ = change wt     ├───────────────┐
     │             │  ←→ = change agent  │               │
     │             │  Enter = terminal   │               │
     │             └──────────┬──────────┘               │
     │                        │                          │
     │                      Enter                    Ctrl+B
     │                        │                          │
     │             ┌──────────▼──────────┐    ┌──────────▼──────────┐
     │             │  TERMINAL MODE      │    │  COMMAND MODE       │
     │             │                     │    │                     │
     │             │  All keys → PTY     │    │  textinput active   │
     │             │  Escape → Navigate  │    │  Enter = execute    │
     │             │                     │    │  Escape = cancel    │
     │             └─────────────────────┘    └─────────────────────┘
     │
     │  (Ctrl+B commands always available from Navigate mode)
     └── Ctrl+B+N: new worktree
         Ctrl+B+A: new agent in current worktree
         Ctrl+B+S: sync current worktree
         Ctrl+B+D: destroy current worktree
         Ctrl+B+F: freeze/unfreeze
         Ctrl+B+Q: quit all
```

---

## 6. UI Layout & Navigation Model

### 6.1 Layout Calculation

```go
// ui/view.go
//
// Total width W, total height H
//
// ┌─────────────────────────────────────────────┐  row 0
// │ title bar                                   │  height: 1
// ├──────────────┬──────────────────────────────┤  row 1
// │              │  agent tab bar               │  height: 1
// │  sidebar     ├──────────────────────────────┤  row 2
// │              │                              │
// │  W*0.25      │  terminal area               │
// │  (min 20)    │  W*0.75                      │
// │              │  height: H - 4               │
// │              │                              │
// ├──────────────┴──────────────────────────────┤  row H-1
// │ status bar                                  │  height: 1
// └─────────────────────────────────────────────┘
//
// Sidebar internal layout:
//   - Worktree list items: variable
//   - Separator line
//   - Metadata block: ~6 lines (branch, agent, gate, status, flags)

const (
    titleBarHeight  = 1
    agentTabsHeight = 1
    statusBarHeight = 1
    fixedRows       = titleBarHeight + agentTabsHeight + statusBarHeight
    sidebarRatio    = 0.25
    sidebarMinWidth = 20
)

func sidebarWidth(totalW int) int {
    w := int(float64(totalW) * sidebarRatio)
    if w < sidebarMinWidth {
        return sidebarMinWidth
    }
    return w
}

func terminalWidth(totalW int) int {
    return totalW - sidebarWidth(totalW) - 1 // -1 for border
}

func terminalHeight(totalH int) int {
    return totalH - fixedRows
}
```

### 6.2 Focus Ring

The sidebar and terminal area share focus:

```
Navigate mode:
  - sidebar has focus → ↑↓ keys move worktree selection
  - Tab key shifts focus to agent tabs
  - agent tabs focused → ←→ keys cycle agents
  - Enter from agent tabs → Terminal mode

Terminal mode:
  - terminal has focus → all keys forwarded to PTY
  - Escape → back to Navigate mode (sidebar focused)
```

---

## 7. Core Package: `worktree/`

```go
// worktree/worktree.go
package worktree

import (
    "encoding/json"
    "os"
    "path/filepath"
    "time"

    "github.com/yourusername/grove/git"
)

type Status string
type GateStatus string

const (
    StatusActive    Status = "active"
    StatusFrozen    Status = "frozen"
    StatusReview    Status = "review"
    StatusMerged    Status = "merged"
    StatusCreated   Status = "created"
    StatusAbandoned Status = "abandoned"

    GateNone     GateStatus = "none"
    GatePending  GateStatus = "pending"
    GateRequired GateStatus = "required"
    GateApproved GateStatus = "approved"
)

type Worktree struct {
    // Identity
    Slug   string `json:"slug"`
    Path   string `json:"path"`
    Branch string `json:"branch"`

    // State (synced with .agent/state.json)
    Status        Status     `json:"status"`
    AssignedAgent string     `json:"assigned_agent"`
    HumanGate     GateStatus `json:"human_gate"`
    Flags         []string   `json:"flags,omitempty"`
    TaskFile      string     `json:"task_file"`
    LastCommit    string     `json:"last_commit,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
    FreezeReason  string     `json:"freeze_reason,omitempty"`

    // Runtime-only (not persisted)
    IsFrozenOnDisk bool `json:"-"` // .agent-frozen file present
}

// LoadAll reads from `git worktree list` and merges with state.json.
// The git output is authoritative for path/branch; state.json adds metadata.
func LoadAll(bareRepo string, stateDir string) ([]Worktree, error) {
    gitWTs, err := git.ListWorktrees(bareRepo)
    if err != nil {
        return nil, err
    }

    stateWTs := loadStateWorktrees(stateDir)

    var result []Worktree
    for _, gwt := range gitWTs {
        wt := Worktree{
            Path:   gwt.Path,
            Branch: gwt.Branch,
            Slug:   slugFromPath(gwt.Path),
        }
        // Merge state.json metadata if present
        if meta, ok := stateWTs[wt.Slug]; ok {
            wt.Status = meta.Status
            wt.AssignedAgent = meta.AssignedAgent
            wt.HumanGate = meta.HumanGate
            wt.Flags = meta.Flags
            wt.TaskFile = meta.TaskFile
            wt.LastCommit = meta.LastCommit
            wt.CreatedAt = meta.CreatedAt
            wt.FreezeReason = meta.FreezeReason
        } else {
            wt.Status = StatusCreated
            wt.HumanGate = GateNone
            wt.CreatedAt = time.Now()
        }
        // Check for .agent-frozen file
        frozenPath := filepath.Join(wt.Path, ".agent-frozen")
        if _, err := os.Stat(frozenPath); err == nil {
            wt.IsFrozenOnDisk = true
        }
        result = append(result, wt)
    }
    return result, nil
}

func slugFromPath(path string) string {
    return filepath.Base(path)
}

// HasFlag checks if a worktree has a specific flag.
func (wt *Worktree) HasFlag(flag string) bool {
    for _, f := range wt.Flags {
        if f == flag {
            return true
        }
    }
    return false
}

// IsSecurityCritical returns true if any security flag is set.
func (wt *Worktree) IsSecurityCritical() bool {
    critical := []string{"security_critical", "touches_auth", "touches_phi", "touches_billing"}
    for _, f := range critical {
        if wt.HasFlag(f) {
            return true
        }
    }
    return false
}

// stateWorktreeJSON mirrors the state.json worktree schema.
type stateWorktreeJSON struct {
    Status        Status     `json:"status"`
    AssignedAgent string     `json:"assigned_agent"`
    HumanGate     GateStatus `json:"human_gate"`
    Flags         []string   `json:"flags"`
    TaskFile      string     `json:"task_file"`
    LastCommit    string     `json:"last_commit"`
    CreatedAt     time.Time  `json:"created_at"`
    FreezeReason  string     `json:"freeze_reason"`
}

type stateJSON struct {
    Worktrees map[string]stateWorktreeJSON `json:"worktrees"`
}

func loadStateWorktrees(stateDir string) map[string]Worktree {
    result := make(map[string]Worktree)
    data, err := os.ReadFile(filepath.Join(stateDir, "state.json"))
    if err != nil {
        return result
    }
    var s stateJSON
    if err := json.Unmarshal(data, &s); err != nil {
        return result
    }
    for slug, meta := range s.Worktrees {
        result[slug] = Worktree{
            Status:        meta.Status,
            AssignedAgent: meta.AssignedAgent,
            HumanGate:     meta.HumanGate,
            Flags:         meta.Flags,
            TaskFile:      meta.TaskFile,
            LastCommit:    meta.LastCommit,
            CreatedAt:     meta.CreatedAt,
            FreezeReason:  meta.FreezeReason,
        }
    }
    return result
}
```

---

## 8. Core Package: `agent/`

```go
// agent/agent.go
package agent

import (
    "fmt"
    "os"
    "path/filepath"
)

type AgentKind string

const (
    Hermes    AgentKind = "hermes"
    ClaudeCode AgentKind = "claude-code"
    Codex     AgentKind = "codex"
    OpenCode  AgentKind = "opencode"
)

type Agent struct {
    Name    AgentKind
    Display string   // for UI labels
    Color   string   // hex color for tab styling
    Command string   // base command
    Args    []string // default args
}

// Registry is the known set of agents.
var Registry = []Agent{
    {
        Name:    ClaudeCode,
        Display: "claude-code",
        Color:   "#D97706", // amber
        Command: "claude",
        Args:    []string{},
    },
    {
        Name:    Codex,
        Display: "codex",
        Color:   "#059669", // emerald
        Command: "codex",
        Args:    []string{},
    },
    {
        Name:    Hermes,
        Display: "hermes",
        Color:   "#7C3AED", // violet
        Command: "hermes",
        Args:    []string{},
    },
    {
        Name:    OpenCode,
        Display: "opencode",
        Color:   "#0891B2", // cyan
        Command: "opencode",
        Args:    []string{},
    },
}

// BuildEnv constructs the environment for a session.
// It injects agent-specific env vars so the agent knows its context.
func BuildEnv(kind AgentKind, worktreePath, taskFile, logFile, agentDir string) []string {
    env := os.Environ()
    env = append(env,
        fmt.Sprintf("GROVE_AGENT=%s", kind),
        fmt.Sprintf("GROVE_WORKTREE=%s", worktreePath),
        fmt.Sprintf("GROVE_TASK_FILE=%s", taskFile),
        fmt.Sprintf("GROVE_LOG_FILE=%s", logFile),
        fmt.Sprintf("GROVE_AGENT_DIR=%s", agentDir),
        fmt.Sprintf("GROVE_MEMORY_DIR=%s", filepath.Join(agentDir, "memory")),
        // Force color output in agent CLIs
        "FORCE_COLOR=1",
        "TERM=xterm-256color",
    )
    return env
}

// Find returns the agent definition by name.
func Find(name AgentKind) (Agent, bool) {
    for _, a := range Registry {
        if a.Name == name {
            return a, true
        }
    }
    return Agent{}, false
}

// AgentColor returns the hex color for an agent (for tab styling).
func AgentColor(name string) string {
    for _, a := range Registry {
        if string(a.Name) == name {
            return a.Color
        }
    }
    return "#888888"
}
```

---

## 9. Core Package: `session/`

This extends the openharness session pattern with worktree context and agent metadata.

```go
// session/session.go
package session

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "time"

    "github.com/creack/pty"
    "github.com/google/uuid"
    "github.com/yourusername/grove/agent"
)

// Session represents a live PTY-backed agent session within a worktree.
type Session struct {
    ID           string
    AgentName    string
    AgentDisplay string
    AgentColor   string
    WorktreeSlug string

    Cmd    *exec.Cmd
    PTY    *os.File
    Output []byte
    Done   bool

    StartedAt time.Time
}

// Spawn creates and starts a new PTY session for an agent in a worktree.
func Spawn(a agent.Agent, worktreePath, agentDir string) (*Session, error) {
    slug := filepath.Base(worktreePath)
    taskFile := filepath.Join(agentDir, "tasks", slug+".yaml")
    logFile := filepath.Join(agentDir, "logs", string(a.Name)+".log")

    cmd := exec.Command(a.Command, a.Args...)
    cmd.Dir = worktreePath
    cmd.Env = agent.BuildEnv(a.Name, worktreePath, taskFile, logFile, agentDir)

    ptmx, err := pty.Start(cmd)
    if err != nil {
        return nil, fmt.Errorf("pty.Start(%s): %w", a.Command, err)
    }

    s := &Session{
        ID:           uuid.New().String(),
        AgentName:    string(a.Name),
        AgentDisplay: a.Display,
        AgentColor:   a.Color,
        WorktreeSlug: slug,
        Cmd:          cmd,
        PTY:          ptmx,
        StartedAt:    time.Now(),
    }

    return s, nil
}

// Close terminates the session gracefully.
func (s *Session) Close() {
    if s.PTY != nil {
        s.PTY.Close()
    }
    if s.Cmd != nil && s.Cmd.Process != nil {
        s.Cmd.Process.Signal(os.Interrupt)
        time.AfterFunc(2*time.Second, func() {
            if !s.Done {
                s.Cmd.Process.Kill()
            }
        })
    }
    s.Done = true
}

// Resize propagates a terminal resize to the PTY.
func (s *Session) Resize(rows, cols int) {
    if s.PTY != nil && !s.Done {
        pty.Setsize(s.PTY, &pty.Winsize{
            Rows: uint16(rows),
            Cols: uint16(cols),
        })
    }
}

// AppendOutput appends bytes to the scrollback buffer, capping at 512KB.
func (s *Session) AppendOutput(data []byte) {
    const maxBuf = 512 * 1024
    s.Output = append(s.Output, data...)
    if len(s.Output) > maxBuf {
        // Trim the oldest half
        s.Output = s.Output[len(s.Output)/2:]
    }
}
```

---

## 10. Core Package: `git/`

```go
// git/git.go
package git

import (
    "bufio"
    "fmt"
    "os/exec"
    "strings"
)

// WorktreeInfo is the parsed output of `git worktree list --porcelain`.
type WorktreeInfo struct {
    Path   string
    HEAD   string
    Branch string
    IsBare bool
    IsMain bool
}

// ListWorktrees returns all worktrees from the bare repo.
func ListWorktrees(bareRepo string) ([]WorktreeInfo, error) {
    cmd := exec.Command("git", "-C", bareRepo, "worktree", "list", "--porcelain")
    out, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("git worktree list: %w", err)
    }
    return parseWorktreeList(string(out)), nil
}

func parseWorktreeList(output string) []WorktreeInfo {
    var worktrees []WorktreeInfo
    var current WorktreeInfo
    scanner := bufio.NewScanner(strings.NewReader(output))

    for scanner.Scan() {
        line := scanner.Text()
        switch {
        case strings.HasPrefix(line, "worktree "):
            if current.Path != "" {
                worktrees = append(worktrees, current)
            }
            current = WorktreeInfo{Path: strings.TrimPrefix(line, "worktree ")}
        case strings.HasPrefix(line, "HEAD "):
            current.HEAD = strings.TrimPrefix(line, "HEAD ")
        case strings.HasPrefix(line, "branch "):
            // "branch refs/heads/feat/auth-mfa" → "feat/auth-mfa"
            ref := strings.TrimPrefix(line, "branch refs/heads/")
            current.Branch = ref
        case line == "bare":
            current.IsBare = true
        }
    }
    if current.Path != "" && !current.IsBare {
        worktrees = append(worktrees, current)
    }
    return worktrees
}

// AddWorktree creates a new worktree + branch.
func AddWorktree(bareRepo, branchName, path, baseBranch string) error {
    cmd := exec.Command("git", "-C", bareRepo,
        "worktree", "add", "-b", branchName, path, baseBranch)
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("git worktree add: %w\n%s", err, out)
    }
    return nil
}

// RemoveWorktree removes a worktree (force).
func RemoveWorktree(bareRepo, path string) error {
    cmd := exec.Command("git", "-C", bareRepo,
        "worktree", "remove", path, "--force")
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("git worktree remove: %w\n%s", err, out)
    }
    return nil
}

// PruneWorktrees removes stale worktree references.
func PruneWorktrees(bareRepo string) error {
    cmd := exec.Command("git", "-C", bareRepo, "worktree", "prune")
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("git worktree prune: %w\n%s", err, out)
    }
    return nil
}

// CurrentBranch returns the current branch of a worktree path.
func CurrentBranch(worktreePath string) (string, error) {
    cmd := exec.Command("git", "-C", worktreePath,
        "symbolic-ref", "--short", "HEAD")
    out, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(out)), nil
}

// LastCommitHash returns the short hash of HEAD.
func LastCommitHash(worktreePath string) string {
    cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--short", "HEAD")
    out, _ := cmd.Output()
    return strings.TrimSpace(string(out))
}

// HasUncommittedChanges returns true if the worktree has uncommitted changes.
func HasUncommittedChanges(worktreePath string) bool {
    cmd := exec.Command("git", "-C", worktreePath, "status", "--porcelain")
    out, err := cmd.Output()
    if err != nil {
        return false
    }
    return len(strings.TrimSpace(string(out))) > 0
}
```

---

## 11. UI Package: `ui/model.go`

```go
// ui/model.go
package ui

import (
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/yourusername/grove/config"
    "github.com/yourusername/grove/session"
    "github.com/yourusername/grove/worktree"
)

// AppMode controls which component owns keyboard input.
type AppMode int

const (
    ModeNavigate AppMode = iota // Grove UI owns keys
    ModeTerminal                // Active PTY owns all keys
    ModeCommand                 // Text input widget is active
)

// LeaderState tracks progress through the Ctrl+B command system.
type LeaderState int

const (
    LeaderNone    LeaderState = iota
    LeaderActive              // Ctrl+B pressed, modal shown
    LeaderConfirm             // Destructive action awaiting y/n
)

// FocusTarget tracks which panel has focus in Navigate mode.
type FocusTarget int

const (
    FocusSidebar    FocusTarget = iota // Worktree list
    FocusAgentTabs                     // Agent tab bar
)

// CommandKind identifies what command is being entered.
type CommandKind int

const (
    CmdNone CommandKind = iota
    CmdNewWorktreeType
    CmdNewWorktreeScope
    CmdNewWorktreeBase
    CmdNewAgentPick
    CmdConfirmDestroy
    CmdConfirmFreeze
)

// Model is the complete application state. All rendering and event handling
// derives from this struct. It must be copyable (value semantics) per BubbleTea.
type Model struct {
    // ── Layout ──────────────────────────────────────────────────────────────
    width  int
    height int

    // ── Navigation ──────────────────────────────────────────────────────────
    mode        AppMode
    focus       FocusTarget
    leader      LeaderState
    leaderPending string // what command is being confirmed

    // ── Worktrees ───────────────────────────────────────────────────────────
    worktrees        []worktree.Worktree
    activeWTIdx      int // index into worktrees slice
    activeSessionIdx map[int]int // worktreeIdx → active session index

    // ── Sessions ────────────────────────────────────────────────────────────
    // sessions[worktreeIdx] → slice of open PTY sessions for that worktree
    sessions map[int][]*session.Session

    // ── Command input ────────────────────────────────────────────────────────
    cmdKind  CommandKind
    cmdInput textinput.Model
    cmdBuf   map[string]string // multi-step command accumulator

    // ── Config ──────────────────────────────────────────────────────────────
    cfg *config.Config

    // ── Notifications ────────────────────────────────────────────────────────
    notification     string
    notificationKind string // "info" | "warn" | "error"
}

func New(cfg *config.Config, worktrees []worktree.Worktree) Model {
    ti := textinput.New()
    ti.CharLimit = 64

    m := Model{
        cfg:              cfg,
        worktrees:        worktrees,
        activeWTIdx:      0,
        activeSessionIdx: make(map[int]int),
        sessions:         make(map[int][]*session.Session),
        mode:             ModeNavigate,
        focus:            FocusSidebar,
        leader:           LeaderNone,
        cmdInput:         ti,
        cmdBuf:           make(map[string]string),
    }
    return m
}

func (m Model) Init() tea.Cmd {
    return nil
}

// ── Convenience helpers ──────────────────────────────────────────────────────

func (m *Model) activeWorktree() *worktree.Worktree {
    if m.activeWTIdx < 0 || m.activeWTIdx >= len(m.worktrees) {
        return nil
    }
    return &m.worktrees[m.activeWTIdx]
}

func (m *Model) activeSessions() []*session.Session {
    return m.sessions[m.activeWTIdx]
}

func (m *Model) activeSession() *session.Session {
    sessions := m.activeSessions()
    idx, ok := m.activeSessionIdx[m.activeWTIdx]
    if !ok || idx >= len(sessions) {
        return nil
    }
    return sessions[idx]
}

func (m *Model) sessionCount() int {
    return len(m.activeSessions())
}

func (m *Model) addSession(worktreeIdx int, s *session.Session) {
    m.sessions[worktreeIdx] = append(m.sessions[worktreeIdx], s)
    m.activeSessionIdx[worktreeIdx] = len(m.sessions[worktreeIdx]) - 1
}

func (m *Model) removeSession(worktreeIdx int, sessionID string) bool {
    sessions := m.sessions[worktreeIdx]
    for i, s := range sessions {
        if s.ID == sessionID {
            s.Close()
            m.sessions[worktreeIdx] = append(sessions[:i], sessions[i+1:]...)
            // Adjust active index
            if m.activeSessionIdx[worktreeIdx] >= len(m.sessions[worktreeIdx]) {
                m.activeSessionIdx[worktreeIdx] = max(0, len(m.sessions[worktreeIdx])-1)
            }
            return true
        }
    }
    return false
}

func (m *Model) setNotification(kind, msg string) {
    m.notification = msg
    m.notificationKind = kind
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
```

---

## 12. UI Package: `ui/messages.go`

```go
// ui/messages.go
package ui

// All tea.Msg types used in the application.

// ptyOutputMsg carries bytes read from a PTY session.
type ptyOutputMsg struct {
    sessionID    string
    worktreeIdx  int
    data         []byte
}

// sessionDoneMsg fires when a PTY session process exits.
type sessionDoneMsg struct {
    sessionID   string
    worktreeIdx int
}

// worktreesRefreshedMsg carries a freshly loaded worktree list.
type worktreesRefreshedMsg struct {
    worktrees []worktree.Worktree
}

// gitOpDoneMsg fires after a git operation completes.
type gitOpDoneMsg struct {
    op  string // "add" | "remove" | "sync"
    err error
}

// notificationMsg sets a timed notification in the status bar.
type notificationMsg struct {
    kind string // "info" | "warn" | "error"
    text string
}
```

---

## 13. UI Package: `ui/update.go`

This is the core of the application — the full state machine.

```go
// ui/update.go
package ui

import (
    "path/filepath"
    "strings"

    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/yourusername/grove/agent"
    "github.com/yourusername/grove/git"
    "github.com/yourusername/grove/session"
    "github.com/yourusername/grove/worktree"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    // ── Window resize ─────────────────────────────────────────────────────
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.propagateResize()
        return m, nil

    // ── PTY output ────────────────────────────────────────────────────────
    case ptyOutputMsg:
        sessions := m.sessions[msg.worktreeIdx]
        for _, s := range sessions {
            if s.ID == msg.sessionID {
                s.AppendOutput(msg.data)
                break
            }
        }
        // Re-issue the read command for this session
        ptmx := m.findPTY(msg.sessionID, msg.worktreeIdx)
        if ptmx != nil {
            return m, waitForOutput(msg.sessionID, msg.worktreeIdx, ptmx)
        }
        return m, nil

    // ── Session process exited ────────────────────────────────────────────
    case sessionDoneMsg:
        sessions := m.sessions[msg.worktreeIdx]
        for _, s := range sessions {
            if s.ID == msg.sessionID {
                s.Done = true
                break
            }
        }
        return m, nil

    // ── Git operations complete ───────────────────────────────────────────
    case gitOpDoneMsg:
        if msg.err != nil {
            m.setNotification("error", "git: "+msg.err.Error())
        } else {
            m.setNotification("info", "git: "+msg.op+" completed")
        }
        // Refresh worktree list
        return m, m.refreshWorktrees()

    // ── Worktrees refreshed ───────────────────────────────────────────────
    case worktreesRefreshedMsg:
        m.worktrees = msg.worktrees
        return m, nil

    // ── Keyboard input ────────────────────────────────────────────────────
    case tea.KeyMsg:
        return m.handleKey(msg)
    }

    return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // Command mode: text input has priority
    if m.mode == ModeCommand {
        return m.handleCommandInput(msg)
    }

    // Ctrl+B leader is always available in Navigate mode
    if msg.Type == tea.KeyCtrlB && m.mode == ModeNavigate {
        m.leader = LeaderActive
        return m, nil
    }

    // If leader is active, handle leader commands
    if m.leader == LeaderActive {
        return m.handleLeaderCommand(msg)
    }

    // Terminal mode: forward everything to PTY
    if m.mode == ModeTerminal {
        // Escape always returns to Navigate mode
        if msg.Type == tea.KeyEscape {
            m.mode = ModeNavigate
            m.focus = FocusSidebar
            return m, nil
        }
        return m.handleTerminalPassthrough(msg)
    }

    // Navigate mode
    return m.handleNavigate(msg)
}

func (m Model) handleNavigate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch m.focus {
    case FocusSidebar:
        return m.handleSidebarNav(msg)
    case FocusAgentTabs:
        return m.handleTabNav(msg)
    }
    return m, nil
}

func (m Model) handleSidebarNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.Type {
    case tea.KeyUp:
        if m.activeWTIdx > 0 {
            m.activeWTIdx--
        }
    case tea.KeyDown:
        if m.activeWTIdx < len(m.worktrees)-1 {
            m.activeWTIdx++
        }
    case tea.KeyTab:
        // Shift focus to agent tabs if sessions exist
        if m.sessionCount() > 0 {
            m.focus = FocusAgentTabs
        }
    case tea.KeyEnter:
        // Enter terminal mode for active session
        if s := m.activeSession(); s != nil && !s.Done {
            m.mode = ModeTerminal
        }
    }
    return m, nil
}

func (m Model) handleTabNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.Type {
    case tea.KeyLeft:
        idx := m.activeSessionIdx[m.activeWTIdx]
        if idx > 0 {
            m.activeSessionIdx[m.activeWTIdx] = idx - 1
        }
    case tea.KeyRight:
        idx := m.activeSessionIdx[m.activeWTIdx]
        sessions := m.activeSessions()
        if idx < len(sessions)-1 {
            m.activeSessionIdx[m.activeWTIdx] = idx + 1
        }
    case tea.KeyEnter:
        if s := m.activeSession(); s != nil && !s.Done {
            m.mode = ModeTerminal
        }
    case tea.KeyEscape, tea.KeyShiftTab:
        m.focus = FocusSidebar
    }
    return m, nil
}

func (m Model) handleTerminalPassthrough(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    s := m.activeSession()
    if s == nil || s.Done {
        m.mode = ModeNavigate
        return m, nil
    }
    raw := keyToBytes(msg)
    if len(raw) > 0 {
        s.PTY.Write(raw)
    }
    return m, nil
}

func (m Model) handleLeaderCommand(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    m.leader = LeaderNone // reset after any key

    if msg.Type != tea.KeyRunes {
        return m, nil
    }

    switch msg.Runes[0] {
    case 'n': // New worktree
        return m.startNewWorktreeFlow()
    case 'a': // New agent session in current worktree
        return m.startNewAgentFlow()
    case 's': // Sync current worktree
        return m, m.syncCurrentWorktree()
    case 'd': // Destroy current worktree
        return m.startDestroyFlow()
    case 'f': // Freeze/unfreeze current worktree
        return m, m.toggleFreeze()
    case 'x': // Close active agent session
        return m.closeActiveSession()
    case 'r': // Refresh worktree list
        return m, m.refreshWorktrees()
    case 'q': // Quit
        return m.shutdown()
    }
    return m, nil
}

// ── New worktree flow ──────────────────────────────────────────────────────

func (m Model) startNewWorktreeFlow() (Model, tea.Cmd) {
    m.mode = ModeCommand
    m.cmdKind = CmdNewWorktreeType
    m.cmdBuf = make(map[string]string)
    m.cmdInput.Reset()
    m.cmdInput.Placeholder = "type: feat|fix|exp|audit|refactor|chore"
    m.cmdInput.Focus()
    return m, textinput.Blink
}

func (m Model) handleCommandInput(msg tea.KeyMsg) (Model, tea.Cmd) {
    switch msg.Type {
    case tea.KeyEscape:
        m.mode = ModeNavigate
        m.cmdKind = CmdNone
        m.cmdInput.Blur()
        return m, nil
    case tea.KeyEnter:
        return m.handleCommandSubmit()
    }
    var cmd tea.Cmd
    m.cmdInput, cmd = m.cmdInput.Update(msg)
    return m, cmd
}

func (m Model) handleCommandSubmit() (Model, tea.Cmd) {
    val := strings.TrimSpace(m.cmdInput.Value())
    m.cmdInput.Reset()

    switch m.cmdKind {
    case CmdNewWorktreeType:
        m.cmdBuf["type"] = val
        m.cmdKind = CmdNewWorktreeScope
        m.cmdInput.Placeholder = "scope (e.g. auth-mfa, rls-policy)"
        return m, textinput.Blink

    case CmdNewWorktreeScope:
        m.cmdBuf["scope"] = val
        m.cmdKind = CmdNewWorktreeBase
        m.cmdInput.Placeholder = "base branch (default: develop)"
        return m, textinput.Blink

    case CmdNewWorktreeBase:
        base := val
        if base == "" {
            base = "develop"
        }
        m.cmdBuf["base"] = base
        m.mode = ModeNavigate
        m.cmdKind = CmdNone
        m.cmdInput.Blur()
        // Execute the git operation in a goroutine
        return m, m.createWorktree(m.cmdBuf["type"], m.cmdBuf["scope"], m.cmdBuf["base"])

    case CmdNewAgentPick:
        // val should be agent name
        m.mode = ModeNavigate
        m.cmdKind = CmdNone
        m.cmdInput.Blur()
        return m, m.spawnAgent(val)

    case CmdConfirmDestroy:
        m.mode = ModeNavigate
        m.cmdKind = CmdNone
        m.cmdInput.Blur()
        if val == "y" {
            return m, m.destroyCurrentWorktree()
        }
    }
    return m, nil
}

// ── New agent flow ────────────────────────────────────────────────────────

func (m Model) startNewAgentFlow() (Model, tea.Cmd) {
    if m.activeWorktree() == nil {
        return m, nil
    }
    m.mode = ModeCommand
    m.cmdKind = CmdNewAgentPick
    m.cmdInput.Reset()
    m.cmdInput.Placeholder = "agent: claude-code|codex|hermes|opencode"
    m.cmdInput.Focus()
    return m, textinput.Blink
}

func (m Model) spawnAgent(agentName string) tea.Cmd {
    return func() tea.Msg {
        wt := m.worktrees[m.activeWTIdx]
        a, ok := agent.Find(agent.AgentKind(agentName))
        if !ok {
            return notificationMsg{kind: "error", text: "unknown agent: " + agentName}
        }
        s, err := session.Spawn(a, wt.Path, m.cfg.AgentDir)
        if err != nil {
            return notificationMsg{kind: "error", text: "spawn failed: " + err.Error()}
        }
        // This msg type needs to add the session to the model
        return spawnedSessionMsg{worktreeIdx: m.activeWTIdx, session: s}
    }
}

// ── Git operations ────────────────────────────────────────────────────────

func (m Model) createWorktree(wtType, scope, base string) tea.Cmd {
    return func() tea.Msg {
        slug := wtType + "-" + scope
        branch := wtType + "/" + scope
        path := filepath.Join(m.cfg.WorktreeBase, slug)
        err := git.AddWorktree(m.cfg.BareRepo, branch, path, base)
        return gitOpDoneMsg{op: "add:" + slug, err: err}
    }
}

func (m Model) syncCurrentWorktree() tea.Cmd {
    wt := m.activeWorktree()
    if wt == nil {
        return nil
    }
    return func() tea.Msg {
        // Run git fetch + rebase in a subprocess
        // (simplified — production version should stream output)
        err := syncWorktreeBranch(wt.Path, m.cfg.BareRepo, wt.Branch)
        return gitOpDoneMsg{op: "sync:" + wt.Slug, err: err}
    }
}

func (m Model) destroyCurrentWorktree() tea.Cmd {
    wt := m.activeWorktree()
    if wt == nil {
        return nil
    }
    return func() tea.Msg {
        // Close all sessions for this worktree first
        err := git.RemoveWorktree(m.cfg.BareRepo, wt.Path)
        return gitOpDoneMsg{op: "remove:" + wt.Slug, err: err}
    }
}

func (m Model) toggleFreeze() tea.Cmd {
    wt := m.activeWorktree()
    if wt == nil {
        return nil
    }
    if wt.IsFrozenOnDisk {
        return unfreezeWorktree(wt.Path)
    }
    return freezeWorktree(wt.Path, "manual freeze via grove")
}

// ── Shutdown ──────────────────────────────────────────────────────────────

func (m Model) shutdown() (Model, tea.Cmd) {
    for _, sessions := range m.sessions {
        for _, s := range sessions {
            s.Close()
        }
    }
    return m, tea.Quit
}

func (m Model) closeActiveSession() (Model, tea.Cmd) {
    s := m.activeSession()
    if s == nil {
        return m, nil
    }
    m.removeSession(m.activeWTIdx, s.ID)
    if m.sessionCount() == 0 {
        m.focus = FocusSidebar
    }
    return m, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────

func (m Model) refreshWorktrees() tea.Cmd {
    return func() tea.Msg {
        wts, err := worktree.LoadAll(m.cfg.BareRepo, m.cfg.AgentDir)
        if err != nil {
            return notificationMsg{kind: "error", text: "refresh: " + err.Error()}
        }
        return worktreesRefreshedMsg{worktrees: wts}
    }
}

func (m *Model) propagateResize() {
    tabBarH := 1
    titleH := 1
    statusH := 1
    ptyH := m.height - tabBarH - titleH - statusH
    ptyW := terminalWidth(m.width)
    if ptyH < 1 {
        ptyH = 1
    }
    for _, sessions := range m.sessions {
        for _, s := range sessions {
            s.Resize(ptyH, ptyW)
        }
    }
}

func (m *Model) findPTY(sessionID string, worktreeIdx int) *os.File {
    for _, s := range m.sessions[worktreeIdx] {
        if s.ID == sessionID && !s.Done {
            return s.PTY
        }
    }
    return nil
}

// waitForOutput is the goroutine bridge: reads PTY bytes and sends a tea.Msg.
func waitForOutput(sessionID string, worktreeIdx int, ptmx *os.File) tea.Cmd {
    return func() tea.Msg {
        buf := make([]byte, 4096)
        n, err := ptmx.Read(buf)
        if err != nil {
            return sessionDoneMsg{sessionID: sessionID, worktreeIdx: worktreeIdx}
        }
        return ptyOutputMsg{
            sessionID:   sessionID,
            worktreeIdx: worktreeIdx,
            data:        buf[:n],
        }
    }
}
```

---

## 14. UI Package: `ui/view.go`

```go
// ui/view.go
package ui

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/yourusername/grove/agent"
    "github.com/yourusername/grove/worktree"
)

func (m Model) View() string {
    if m.width == 0 {
        return "initializing..."
    }

    // Leader modal overlays everything
    if m.leader == LeaderActive {
        return m.renderLeaderModal()
    }

    // Command mode overlays the bottom portion
    sw := sidebarWidth(m.width)
    tw := terminalWidth(m.width)
    th := terminalHeight(m.height)

    sidebar := m.renderSidebar(sw, th+2) // +2 for title + agent tabs rows
    right := lipgloss.JoinVertical(lipgloss.Left,
        m.renderAgentTabs(tw),
        m.renderTerminal(tw, th),
    )

    body := lipgloss.JoinHorizontal(lipgloss.Top,
        sidebar,
        styles.Divider.Render(""),
        right,
    )

    return lipgloss.JoinVertical(lipgloss.Left,
        m.renderTitleBar(),
        body,
        m.renderStatusBar(),
    )
}

// ── Title bar ─────────────────────────────────────────────────────────────

func (m Model) renderTitleBar() string {
    left := styles.TitleApp.Render(" grove ")

    var middle string
    if wt := m.activeWorktree(); wt != nil {
        middle = styles.TitleWorktree.Render(wt.Slug)
    }

    modeLabel := ""
    switch m.mode {
    case ModeTerminal:
        modeLabel = styles.TitleModeTerminal.Render(" TERMINAL ")
    case ModeCommand:
        modeLabel = styles.TitleModeCommand.Render(" COMMAND ")
    case ModeNavigate:
        modeLabel = styles.TitleModeNavigate.Render(" NAVIGATE ")
    }

    gap := m.width - lipgloss.Width(left) - lipgloss.Width(middle) - lipgloss.Width(modeLabel)
    if gap < 0 {
        gap = 0
    }

    return styles.TitleBar.Width(m.width).Render(
        left + middle + strings.Repeat(" ", gap) + modeLabel,
    )
}

// ── Sidebar ───────────────────────────────────────────────────────────────

func (m Model) renderSidebar(w, h int) string {
    var lines []string

    // Header
    header := styles.SidebarHeader.Width(w).Render(
        fmt.Sprintf(" WORKTREES  [%d]", len(m.worktrees)),
    )
    lines = append(lines, header)

    // Worktree list
    for i, wt := range m.worktrees {
        active := i == m.activeWTIdx

        // Status dot
        dot := statusDot(wt.Status)

        // Session count badge
        sessionBadge := ""
        if n := len(m.sessions[i]); n > 0 {
            sessionBadge = fmt.Sprintf(" [%d]", n)
        }

        label := fmt.Sprintf(" %s %s%s", dot, wt.Slug, sessionBadge)

        if active && m.focus == FocusSidebar {
            lines = append(lines, styles.SidebarItemActive.Width(w).Render(label))
        } else if active {
            lines = append(lines, styles.SidebarItemSelected.Width(w).Render(label))
        } else {
            lines = append(lines, styles.SidebarItem.Width(w).Render(label))
        }
    }

    // Separator + metadata
    if wt := m.activeWorktree(); wt != nil {
        lines = append(lines, styles.SidebarSep.Width(w).Render(strings.Repeat("─", w)))
        lines = append(lines, m.renderMetadata(wt, w)...)
    }

    // Pad to height
    content := strings.Join(lines, "\n")
    contentH := strings.Count(content, "\n") + 1
    if contentH < h {
        content += strings.Repeat("\n"+styles.SidebarItem.Width(w).Render(""), h-contentH)
    }

    return styles.Sidebar.Width(w).Height(h).Render(content)
}

func (m Model) renderMetadata(wt *worktree.Worktree, w int) []string {
    pad := styles.SidebarMeta.Width(w)
    var lines []string

    lines = append(lines,
        pad.Render(" Branch:"),
        pad.Render("  "+truncate(wt.Branch, w-3)),
        pad.Render(fmt.Sprintf(" Agent:  %s", wt.AssignedAgent)),
        pad.Render(fmt.Sprintf(" Gate:   %s", gateLabel(wt.HumanGate))),
        pad.Render(fmt.Sprintf(" Status: %s", wt.Status)),
    )

    if wt.IsSecurityCritical() {
        lines = append(lines, styles.SidebarWarning.Width(w).Render(" ⚠ security"))
    }
    if wt.IsFrozenOnDisk {
        lines = append(lines, styles.SidebarFrozen.Width(w).Render(" ❄ frozen"))
    }

    return lines
}

// ── Agent tabs ────────────────────────────────────────────────────────────

func (m Model) renderAgentTabs(w int) string {
    sessions := m.activeSessions()
    if len(sessions) == 0 {
        hint := " no agents — Ctrl+B+A to spawn"
        return styles.AgentTabBarEmpty.Width(w).Render(hint)
    }

    activeIdx := m.activeSessionIdx[m.activeWTIdx]
    var tabs []string

    for i, s := range sessions {
        agentColor := agent.AgentColor(s.AgentName)
        doneMarker := ""
        if s.Done {
            doneMarker = " ✗"
        }
        label := fmt.Sprintf(" %s%s ", s.AgentDisplay, doneMarker)

        focused := i == activeIdx && m.focus == FocusAgentTabs

        style := styles.AgentTab.
            Foreground(lipgloss.Color(agentColor))

        if i == activeIdx {
            style = style.Bold(true).
                Background(lipgloss.Color("#1E1E2E")).
                Underline(focused)
        }

        tabs = append(tabs, style.Render(label))
    }

    return styles.AgentTabBar.Width(w).Render(strings.Join(tabs, ""))
}

// ── Terminal ──────────────────────────────────────────────────────────────

func (m Model) renderTerminal(w, h int) string {
    s := m.activeSession()
    if s == nil {
        empty := styles.TerminalEmpty.Width(w).Height(h)
        return empty.Render(
            "\n\n  No agent running.\n\n" +
                "  Ctrl+B → A  spawn agent\n" +
                "  Ctrl+B → N  new worktree",
        )
    }

    output := string(s.Output)
    lines := strings.Split(output, "\n")
    if len(lines) > h {
        lines = lines[len(lines)-h:]
    }
    // Pad to height
    for len(lines) < h {
        lines = append(lines, "")
    }

    content := strings.Join(lines, "\n")

    style := styles.Terminal.Width(w).Height(h)
    if m.mode == ModeTerminal {
        style = style.BorderForeground(lipgloss.Color("#5C7BC9"))
    }

    return style.Render(content)
}

// ── Status bar ────────────────────────────────────────────────────────────

func (m Model) renderStatusBar() string {
    var left, right string

    if m.mode == ModeCommand {
        left = " " + m.cmdInput.View()
    } else if m.notification != "" {
        notifStyle := styles.NotifInfo
        if m.notificationKind == "error" {
            notifStyle = styles.NotifError
        } else if m.notificationKind == "warn" {
            notifStyle = styles.NotifWarn
        }
        left = notifStyle.Render(" " + m.notification)
    } else {
        switch m.mode {
        case ModeNavigate:
            left = " Ctrl+B: commands  ↑↓: worktrees  Tab: agents  Enter: terminal"
        case ModeTerminal:
            left = " TERMINAL MODE — Escape to exit"
        }
    }

    // Right side: worktree/agent context
    wt := m.activeWorktree()
    if wt != nil {
        ctx := wt.Slug
        if s := m.activeSession(); s != nil {
            ctx += "  " + s.AgentDisplay
        }
        right = styles.StatusBarRight.Render(ctx + " ")
    }

    gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
    if gap < 0 {
        gap = 0
    }

    return styles.StatusBar.Width(m.width).Render(
        left + strings.Repeat(" ", gap) + right,
    )
}

// ── Leader modal ──────────────────────────────────────────────────────────

func (m Model) renderLeaderModal() string {
    lines := []string{
        " grove commands\n",
        " [N]  new worktree",
        " [A]  new agent in current worktree",
        " [S]  sync (rebase) current worktree",
        " [D]  destroy current worktree",
        " [F]  freeze / unfreeze",
        " [X]  close active agent session",
        " [R]  refresh worktree list",
        " [Q]  quit all",
        "",
        " Escape to cancel",
    }
    content := strings.Join(lines, "\n")
    modal := styles.Modal.Render(content)
    return lipgloss.Place(m.width, m.height,
        lipgloss.Center, lipgloss.Center, modal)
}

// ── Helpers ───────────────────────────────────────────────────────────────

func statusDot(s worktree.Status) string {
    switch s {
    case worktree.StatusActive:
        return "●" // green
    case worktree.StatusFrozen:
        return "❄"
    case worktree.StatusReview:
        return "◆" // yellow
    case worktree.StatusMerged:
        return "✓"
    default:
        return "○"
    }
}

func gateLabel(g worktree.GateStatus) string {
    switch g {
    case worktree.GateRequired:
        return "⚠ required"
    case worktree.GatePending:
        return "⏳ pending"
    case worktree.GateApproved:
        return "✓ approved"
    default:
        return "none"
    }
}

func truncate(s string, max int) string {
    if len(s) <= max {
        return s
    }
    return s[:max-1] + "…"
}
```

---

## 15. UI Package: `ui/styles.go`

```go
// ui/styles.go
package styles

import "github.com/charmbracelet/lipgloss"

// ── Color palette (Kanagawa-inspired to match your Neovim theme) ──────────

var (
    colorBg          = lipgloss.Color("#1F1F28")
    colorBgDark      = lipgloss.Color("#16161D")
    colorBgHighlight = lipgloss.Color("#2A2A37")
    colorBorder      = lipgloss.Color("#54546D")
    colorActive      = lipgloss.Color("#7E9CD8") // Kanagawa blue
    colorAmber       = lipgloss.Color("#DCA561") // for warnings
    colorGreen       = lipgloss.Color("#98BB6C")
    colorRed         = lipgloss.Color("#E46876")
    colorMuted       = lipgloss.Color("#727169")
    colorText        = lipgloss.Color("#DCD7BA")
    colorSubtle      = lipgloss.Color("#938AA9")
)

// ── Title bar ─────────────────────────────────────────────────────────────

var TitleBar = lipgloss.NewStyle().
    Background(colorBgDark).
    Foreground(colorText)

var TitleApp = lipgloss.NewStyle().
    Background(colorActive).
    Foreground(colorBgDark).
    Bold(true).
    Padding(0, 1)

var TitleWorktree = lipgloss.NewStyle().
    Background(colorBgDark).
    Foreground(colorSubtle).
    Padding(0, 1)

var TitleModeTerminal = lipgloss.NewStyle().
    Background(colorGreen).
    Foreground(colorBgDark).
    Bold(true)

var TitleModeNavigate = lipgloss.NewStyle().
    Background(colorBgHighlight).
    Foreground(colorMuted)

var TitleModeCommand = lipgloss.NewStyle().
    Background(colorAmber).
    Foreground(colorBgDark).
    Bold(true)

// ── Sidebar ───────────────────────────────────────────────────────────────

var Sidebar = lipgloss.NewStyle().
    Background(colorBgDark).
    BorderRight(true).
    BorderForeground(colorBorder)

var SidebarHeader = lipgloss.NewStyle().
    Background(colorBgDark).
    Foreground(colorMuted).
    Bold(true).
    Padding(0, 1)

var SidebarItem = lipgloss.NewStyle().
    Background(colorBgDark).
    Foreground(colorSubtle).
    Padding(0, 1)

var SidebarItemSelected = lipgloss.NewStyle().
    Background(colorBgDark).
    Foreground(colorText).
    Padding(0, 1)

var SidebarItemActive = lipgloss.NewStyle().
    Background(colorBgHighlight).
    Foreground(colorActive).
    Bold(true).
    Padding(0, 1)

var SidebarSep = lipgloss.NewStyle().
    Foreground(colorBorder)

var SidebarMeta = lipgloss.NewStyle().
    Background(colorBgDark).
    Foreground(colorMuted).
    Padding(0, 1)

var SidebarWarning = lipgloss.NewStyle().
    Background(colorBgDark).
    Foreground(colorAmber)

var SidebarFrozen = lipgloss.NewStyle().
    Background(colorBgDark).
    Foreground(colorActive)

var Divider = lipgloss.NewStyle().
    BorderLeft(true).
    BorderForeground(colorBorder)

// ── Agent tabs ────────────────────────────────────────────────────────────

var AgentTabBar = lipgloss.NewStyle().
    Background(colorBgDark).
    BorderBottom(true).
    BorderForeground(colorBorder)

var AgentTabBarEmpty = lipgloss.NewStyle().
    Background(colorBgDark).
    Foreground(colorMuted).
    Italic(true)

var AgentTab = lipgloss.NewStyle().
    Padding(0, 1)

// ── Terminal ──────────────────────────────────────────────────────────────

var Terminal = lipgloss.NewStyle().
    Background(colorBg).
    Foreground(colorText).
    Border(lipgloss.NormalBorder()).
    BorderForeground(colorBorder)

var TerminalEmpty = lipgloss.NewStyle().
    Background(colorBg).
    Foreground(colorMuted).
    Italic(true)

// ── Status bar ────────────────────────────────────────────────────────────

var StatusBar = lipgloss.NewStyle().
    Background(colorBgDark).
    Foreground(colorMuted)

var StatusBarRight = lipgloss.NewStyle().
    Background(colorBgDark).
    Foreground(colorSubtle)

var NotifInfo = lipgloss.NewStyle().
    Foreground(colorGreen)

var NotifWarn = lipgloss.NewStyle().
    Foreground(colorAmber)

var NotifError = lipgloss.NewStyle().
    Foreground(colorRed)

// ── Modal ─────────────────────────────────────────────────────────────────

var Modal = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(colorActive).
    Background(colorBgDark).
    Foreground(colorText).
    Padding(1, 3)
```

---

## 16. Config System

```go
// config/config.go
package config

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "tailscale.com/util/hujson"
)

type Config struct {
    // Required
    BareRepo     string `json:"bare_repo"`     // e.g. ~/projects/myapp.git
    WorktreeBase string `json:"worktree_base"` // e.g. ~/projects/myapp/wt
    AgentDir     string `json:"agent_dir"`     // e.g. ~/projects/myapp/.agent

    // Agent CLI overrides (optional)
    AgentCommands map[string]string `json:"agent_commands,omitempty"`

    // UI preferences
    SidebarWidth int  `json:"sidebar_width,omitempty"` // 0 = auto (25%)
    MouseEnabled bool `json:"mouse_enabled,omitempty"`
}

func Load() (*Config, error) {
    path := configPath()
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return nil, writeDefaultConfig(path)
    }
    return loadFromFile(path)
}

func loadFromFile(path string) (*Config, error) {
    raw, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    standardized, err := hujson.Standardize(raw)
    if err != nil {
        return nil, fmt.Errorf("invalid JSONC config: %w", err)
    }
    var cfg Config
    if err := json.Unmarshal(standardized, &cfg); err != nil {
        return nil, fmt.Errorf("config parse error: %w", err)
    }
    cfg.BareRepo = expandHome(cfg.BareRepo)
    cfg.WorktreeBase = expandHome(cfg.WorktreeBase)
    cfg.AgentDir = expandHome(cfg.AgentDir)
    return &cfg, nil
}

func configPath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "grove", "config.jsonc")
}

func expandHome(path string) string {
    if len(path) > 1 && path[:2] == "~/" {
        home, _ := os.UserHomeDir()
        return filepath.Join(home, path[2:])
    }
    return path
}

// writeDefaultConfig writes a first-run config and returns an error
// so the caller can tell the user to configure before running.
func writeDefaultConfig(path string) error {
    os.MkdirAll(filepath.Dir(path), 0755)
    const defaultCfg = `{
  // grove config — edit before running
  // See: https://github.com/yourusername/grove

  // Path to your bare git repo
  "bare_repo": "~/projects/myapp.git",

  // Directory where worktrees are checked out
  "worktree_base": "~/projects/myapp/wt",

  // Directory containing .agent/ context store
  "agent_dir": "~/projects/myapp/.agent",

  // Optional: override the CLI command for each agent
  // "agent_commands": {
  //   "claude-code": "claude",
  //   "codex": "codex",
  //   "hermes": "hermes",
  //   "opencode": "opencode"
  // }
}
`
    if err := os.WriteFile(path, []byte(defaultCfg), 0644); err != nil {
        return fmt.Errorf("could not write default config: %w", err)
    }
    return fmt.Errorf("grove: created default config at %s — please configure and re-run", path)
}
```

---

## 17. main.go Entrypoint

```go
// main.go
package main

import (
    "fmt"
    "os"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/yourusername/grove/config"
    "github.com/yourusername/grove/ui"
    "github.com/yourusername/grove/worktree"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    // Load initial worktree list
    worktrees, err := worktree.LoadAll(cfg.BareRepo, cfg.AgentDir)
    if err != nil {
        // Non-fatal: start with empty list, user can refresh
        fmt.Fprintf(os.Stderr, "warning: could not load worktrees: %v\n", err)
        worktrees = nil
    }

    m := ui.New(cfg, worktrees)

    opts := []tea.ProgramOption{
        tea.WithAltScreen(),
    }
    if cfg.MouseEnabled {
        opts = append(opts, tea.WithMouseCellMotion())
    }

    p := tea.NewProgram(m, opts...)
    if _, err := p.Run(); err != nil {
        fmt.Fprintln(os.Stderr, "error:", err)
        os.Exit(1)
    }
}
```

---

## 18. Key Pitfalls & Critical Patterns

These are the non-obvious issues you'll hit building this specific tool on top of the openharness patterns.

### 18.1 The Worktree Index Stability Problem

The `sessions` map uses `int` (worktree index) as a key. **If the worktrees slice is reordered after a refresh, all session-to-worktree associations break.** Fix: use slug (string) as the map key.

```go
// ✗ Wrong — index-based, breaks on refresh
sessions map[int][]*session.Session

// ✓ Correct — slug-based, stable across refreshes
sessions map[string][]*session.Session
```

Update all references in `model.go` accordingly.

### 18.2 PTY Resize Timing

When you call `Spawn()` and then immediately `Resize()`, the PTY process may not have fully initialized yet. The fix is to call `Resize()` in the `spawnedSessionMsg` handler (after the goroutine confirms the process started), not in the `spawnAgent` command.

```go
case spawnedSessionMsg:
    m.addSession(msg.worktreeSlug, msg.session)
    // Resize NOW, after the session is in the model
    ptyH := terminalHeight(m.height)
    ptyW := terminalWidth(m.width)
    msg.session.Resize(ptyH, ptyW)
    // Start reading output
    return m, waitForOutput(msg.session.ID, msg.worktreeSlug, msg.session.PTY)
```

### 18.3 The `tea.Cmd` for `spawnedSessionMsg`

`spawnAgent` returns a `tea.Cmd` that runs in a goroutine. But adding the session to the model has to happen in `Update()`, not in the goroutine. The goroutine returns a message (`spawnedSessionMsg`) that carries the `*session.Session` — and `Update` adds it to the model.

```go
type spawnedSessionMsg struct {
    worktreeSlug string
    session      *session.Session
}
```

### 18.4 Concurrent Output from Multiple Agents

If two agents across different worktrees are running simultaneously, their `waitForOutput` goroutines both send `ptyOutputMsg` to the Bubble Tea program. Bubble Tea processes messages serially, so there's no race. **But:** the `sessionID` + `worktreeSlug` combo in `ptyOutputMsg` must uniquely identify the session across all worktrees — use the UUID from `session.Session.ID`, not just an index.

### 18.5 Mode Reset on Worktree Switch

When the user navigates to a different worktree while in `ModeTerminal`, you must exit terminal mode first. This is a UX choice — implement it in `handleSidebarNav`:

```go
case tea.KeyUp, tea.KeyDown:
    if m.mode == ModeTerminal {
        m.mode = ModeNavigate  // exit terminal mode on wt navigation
    }
    // ... change activeWTIdx
```

### 18.6 Frozen Worktree Guard

If a worktree is frozen (`.agent-frozen` exists), don't allow spawning new agent sessions:

```go
func (m Model) startNewAgentFlow() (Model, tea.Cmd) {
    wt := m.activeWorktree()
    if wt == nil {
        return m, nil
    }
    if wt.IsFrozenOnDisk {
        m.setNotification("warn", "worktree is frozen — unfreeze first (Ctrl+B+F)")
        return m, nil
    }
    // ... proceed
}
```

---

## 19. Build Phases

Build this in five phases. Don't skip ahead — each phase gives you something testable.

### Phase 1: Skeleton + Worktree List (1-2 days)

- [ ] `git/git.go` — `ListWorktrees()` parsing
- [ ] `worktree/worktree.go` — `LoadAll()` (git only, no state.json yet)
- [ ] `config/config.go` — JSONC loading
- [ ] `ui/` — Sidebar renders worktree list; no sessions yet
- [ ] `main.go` — boots, shows sidebar
- **Test:** Can you see your worktrees listed? ↑↓ navigation works?

### Phase 2: PTY Sessions + Agent Tabs (2-3 days)

- [ ] `agent/agent.go` — Registry + `BuildEnv()`
- [ ] `session/session.go` — `Spawn()` + `Close()` + `AppendOutput()`
- [ ] `ui/update.go` — `spawnAgent()`, `waitForOutput()`, `ptyOutputMsg` handler
- [ ] `ui/view.go` — Agent tab bar + terminal area renders PTY output
- [ ] Mode switching: Navigate ↔ Terminal (Escape exits)
- **Test:** Ctrl+B+A, type "claude-code", Enter → does `claude` spawn in the terminal panel?

### Phase 3: Full Navigation + Multi-Agent (1-2 days)

- [ ] Multiple sessions per worktree
- [ ] ←→ tab cycling in `FocusAgentTabs`
- [ ] `closeActiveSession()` (Ctrl+B+X)
- [ ] Session done detection (`sessionDoneMsg`)
- [ ] Resize propagation to all sessions
- **Test:** Spawn claude-code AND codex in the same worktree. Switch between them. Close one.

### Phase 4: Git Operations + State (2-3 days)

- [ ] `git.AddWorktree()` / `git.RemoveWorktree()`
- [ ] New worktree flow (Ctrl+B+N multi-step command input)
- [ ] Destroy flow (Ctrl+B+D with confirmation)
- [ ] Freeze/unfreeze (Ctrl+B+F)
- [ ] Sync worktree (Ctrl+B+S — runs in background goroutine)
- [ ] `state/state.go` — read/write `.agent/state.json`
- [ ] Worktree refresh after git ops
- **Test:** Create a new worktree from inside grove. Does it appear in the sidebar?

### Phase 5: Polish + Notifications (1 day)

- [ ] Timed notification clearing (auto-clear after 3s)
- [ ] Security/gate warnings in sidebar
- [ ] Status dot colors match worktree status
- [ ] Config: `mouse_enabled`, `sidebar_width` override
- [ ] `grove --version` flag
- [ ] README with keybinding table

---

## 20. Keybinding Reference

```
NAVIGATE MODE (default)
─────────────────────────────────────────────────────────
↑ / ↓             Move between worktrees (sidebar focused)
← / →             Cycle agent tabs (tab bar focused)
Tab               Shift focus: sidebar → agent tabs
Shift+Tab         Shift focus: agent tabs → sidebar
Enter             Enter Terminal mode for active agent
Ctrl+B            Activate leader (command modal)

LEADER COMMANDS (after Ctrl+B)
─────────────────────────────────────────────────────────
N                 New worktree (multi-step input)
A                 New agent session in current worktree
S                 Sync / rebase current worktree
D                 Destroy current worktree (confirm required)
F                 Freeze / unfreeze current worktree
X                 Close active agent session
R                 Refresh worktree list from git
Q                 Quit grove (closes all sessions)
Escape            Cancel leader

TERMINAL MODE
─────────────────────────────────────────────────────────
(all keys)        Forwarded to active agent PTY
Escape            Exit Terminal mode → Navigate mode

COMMAND MODE (text input active)
─────────────────────────────────────────────────────────
(typing)          Text input
Enter             Submit current step
Escape            Cancel command, return to Navigate mode
```

---

_grove — built on openharness patterns, extended for the multi-agent worktree workflow_  
_Stack: Go + Bubble Tea v0.27 + Lip Gloss v1 + creack/pty v1.1_