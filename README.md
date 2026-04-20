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

How to use:

1. Start Grove from anywhere:

```bash
grove
```

2. Grove will look for the nearest Git repository by walking up from your current folder.

3. If you are already inside a normal repo, you will see the worktree list right away.

4. If you are inside a bare repo folder or a folder that is not a repo, Grove will show a no-repo screen instead of crashing. In that screen:
   - check the current path shown on screen
   - move into a real git working tree directory and run `grove` again
   - or set `GROVE_ROOT=/path/to/repo` if you want Grove to open a specific repository

5. Once you are in repo mode, these keys work:
   - `r` or `F5`: refresh worktrees
   - `n` or `c`: start creating a new worktree
   - `Enter`: select the highlighted worktree or confirm the create prompt
   - `Esc`: cancel the current prompt
   - `q` or `Ctrl+C`: quit

6. When you choose to create a worktree, Grove will ask for a name. Type the name and press `Enter`.
   - The current build already records the request and reloads the UI
   - full worktree creation/switching workflows are the next major feature layer

Quickstart:

```bash
# If you installed with go install
grove --help
grove

# Or run directly from source
go run ./cmd/grove
```

Environment:
- `GROVE_ROOT`: force a repo root
- `GROVE_CONFIG`: optional config path

Release process:
- tags like `v0.1.0` trigger GitHub Actions releases
- release builds produce archives for Linux, macOS, and Windows
