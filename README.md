# Grove

Grove is a terminal TUI for managing git worktrees and AI agent sessions.

Status:
- builds cleanly
- runs from any folder
- detects the nearest git repo root automatically
- shows a helpful landing screen when no repo is present
- can list and refresh worktrees
- supports a basic create-worktree prompt

Quickstart:

```bash
cd /home/saddat/Documents/Projects/Personal/Golang/grove
go run ./cmd/grove --help
go run ./cmd/grove
```

Useful keys inside the TUI:
- q / Ctrl+C: quit
- r / F5: refresh worktrees
- n or c: create a worktree
- Enter: select/confirm
- Esc: cancel

Environment:
- GROVE_ROOT: force a repo root
- GROVE_CONFIG: optional config path
