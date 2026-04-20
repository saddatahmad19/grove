# Grove

Grove is a terminal TUI for managing git worktrees and AI agent sessions.

Status:
- builds cleanly
- runs from any folder
- detects the nearest git repo root automatically
- shows a helpful landing screen when no repo is present
- can list and refresh worktrees
- supports a basic create-worktree prompt

Installation:

```bash
# Go install for anyone with Go installed
go install github.com/saddatahmad19/grove/cmd/grove@latest

# Or download a release binary from GitHub Releases
# https://github.com/saddatahmad19/grove/releases
```

Quickstart:

```bash
grove --help
grove
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

Release process:
- tags like `v0.1.0` trigger GitHub Actions releases
- release builds produce archives for Linux, macOS, and Windows
