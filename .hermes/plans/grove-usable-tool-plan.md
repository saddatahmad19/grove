# Grove Usable Tool Implementation Plan

> **For Hermes:** Use subagent-driven-development to implement this plan task-by-task.

**Goal:** Turn Grove into a usable terminal tool that can run in any folder, detect whether it is inside a git repo, and guide the user toward worktree/agent actions instead of failing when the context is missing.

**Architecture:** Keep the current Bubble Tea scaffold, but add a real startup state machine with three modes: no repo, repo loaded, and action-ready. Add a proper command/help footer, robust git-root discovery from any folder, and safe fallbacks when no repository is present. Preserve the current compile-first design and avoid overbuilding PTY/session orchestration until the basic user experience is solid.

**Tech Stack:** Go, Bubble Tea, Bubbles list, Lip Gloss, os/exec, filepath, git CLI.

---

### Task 1: Make startup work from any folder

**Objective:** Discover the nearest git repository root from the current working directory, and if none exists, keep the app usable with a clear "no repo" state instead of failing.

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/app/app.go`
- Modify: `internal/ui/model.go`

**Step 1: Write failing test**

Create `internal/config/config_test.go` with tests for:
- loading from a git repo subdirectory returns the repo root
- loading from a non-git directory returns a usable config and no error

Example:
```go
func TestLoadFindsGitRoot(t *testing.T) {}
func TestLoadHandlesNonRepo(t *testing.T) {}
```

**Step 2: Run test to verify failure**

Run: `go test ./internal/config -v`
Expected: FAIL — config.Load does not yet search upward for a .git directory.

**Step 3: Write minimal implementation**

Implement upward directory search using `filepath.Abs`, `filepath.Clean`, and a loop to the filesystem root. If no `.git` is found, return the original working directory with a `RepoFound=false` flag.

Suggested config shape:
```go
type Config struct {
    Root      string
    RepoFound bool
}
```

**Step 4: Run test to verify pass**

Run: `go test ./internal/config -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: detect git root from any folder"
```

### Task 2: Make the TUI explain itself when no repo is present

**Objective:** Show a helpful landing screen with actions and a clear instruction when Grove is launched outside a git repo.

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/view.go`
- Modify: `internal/ui/update.go`
- Modify: `internal/ui/styles.go`

**Step 1: Write failing test**

Create `internal/ui/model_test.go` that asserts:
- `NewModel` with `RepoFound=false` shows a no-repo status
- `View()` includes a short instruction like "Open a git repository to see worktrees"

Example:
```go
func TestViewShowsNoRepoMessage(t *testing.T) {}
```

**Step 2: Run test to verify failure**

Run: `go test ./internal/ui -v`
Expected: FAIL — current view does not distinguish repo vs no-repo.

**Step 3: Write minimal implementation**

Add a landing state and render a concise message with suggested next steps:
- `cd` into a repo
- or set `GROVE_ROOT`
- then restart Grove

**Step 4: Run test to verify pass**

Run: `go test ./internal/ui -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/*
git commit -m "feat: improve no-repo startup experience"
```

### Task 3: Add useful keybindings and footer help

**Objective:** Make the TUI feel like a tool instead of a demo by exposing the most important keys directly on screen.

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/view.go`

**Step 1: Write failing test**

Add tests for a footer helper that renders key hints such as:
- `r` refresh
- `q` quit
- `tab` switch panel
- `?` help

**Step 2: Run test to verify failure**

Run: `go test ./internal/ui -v`
Expected: FAIL

**Step 3: Write minimal implementation**

Render a compact footer with the key hints.

**Step 4: Run test to verify pass**

Run: `go test ./internal/ui -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/view.go internal/ui/model.go
git commit -m "feat: add TUI help footer"
```

### Task 4: Add refresh and basic worktree reload

**Objective:** Let the user refresh the worktree list after switching branches or creating/removing worktrees outside Grove.

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/worktree/worktree.go`
- Modify: `internal/git/git.go`

**Step 1: Write failing test**

Create tests for `worktree.LoadAll` against a repo fixture or a mocked parser so refresh behavior is deterministic.

**Step 2: Run test to verify failure**

Run: `go test ./internal/worktree ./internal/git -v`
Expected: FAIL if parser assumptions are wrong.

**Step 3: Write minimal implementation**

Add a `Refresh()` command triggered by `r` that reloads worktrees and updates status.

**Step 4: Run test to verify pass**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/worktree internal/git internal/ui
git commit -m "feat: add worktree refresh"
```

### Task 5: Document how to use Grove

**Objective:** Make the repo self-explanatory with clear setup and usage notes.

**Files:**
- Modify: `README.md`

**Step 1: Write failing test**

No code test needed; verify via manual review that README mentions:
- build command
- run command
- `GROVE_ROOT`
- no-repo behavior

**Step 2: Run verification**

Run: `go run ./cmd/grove --help`
Expected: prints help

**Step 3: Write minimal implementation**

Expand README with quickstart and behavior notes.

**Step 4: Run verification**

Check README content and run `go test ./...`.

**Step 5: Commit**

```bash
git add README.md
git commit -m "docs: add Grove quickstart"
```

---

## Final verification

After all tasks:

```bash
go test ./...
go build ./...
go run ./cmd/grove --help
```

Expected:
- all tests pass
- build passes
- help text prints successfully

## Notes

- Keep the current scaffold lean.
- Prioritize usability from any working directory before adding advanced agent/session orchestration.
- If a test requires a git repo fixture, create the smallest possible fixture in `testdata/`.
