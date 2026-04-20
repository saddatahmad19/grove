package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Worktree struct {
	Path   string
	Branch string
	Head   string
	Bare   bool
}

type WorktreeCreateOptions struct {
	Path   string
	Branch string
	Start  string
}

func ListWorktrees(root string) ([]Worktree, error) {
	cmd := exec.Command("git", "-C", root, "worktree", "list", "--porcelain")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, commandError("list worktrees", err, out)
	}
	return parseWorktreePorcelain(out), nil
}

func CreateWorktree(root string, opts WorktreeCreateOptions) (string, error) {
	if strings.TrimSpace(opts.Path) == "" {
		return "", errors.New("create worktree: path is required")
	}
	args := []string{"-C", root, "worktree", "add"}
	if strings.TrimSpace(opts.Branch) != "" {
		args = append(args, "-b", opts.Branch)
	}
	args = append(args, opts.Path)
	if strings.TrimSpace(opts.Start) != "" {
		args = append(args, opts.Start)
	}
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", commandError("create worktree", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

func parseWorktreePorcelain(out []byte) []Worktree {
	var result []Worktree
	var cur Worktree
	flush := func() {
		if cur.Path != "" {
			result = append(result, cur)
		}
		cur = Worktree{}
	}
	for _, raw := range bytes.Split(out, []byte("\n")) {
		line := strings.TrimSpace(string(raw))
		if line == "" {
			flush()
			continue
		}
		fields := strings.SplitN(line, " ", 2)
		if len(fields) != 2 {
			continue
		}
		switch fields[0] {
		case "worktree":
			cur.Path = fields[1]
		case "branch":
			cur.Branch = strings.TrimPrefix(fields[1], "refs/heads/")
		case "HEAD":
			cur.Head = fields[1]
		case "bare":
			cur.Bare = fields[1] == "true"
		}
	}
	flush()
	return result
}

func commandError(action string, err error, out []byte) error {
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		return fmt.Errorf("%s: %w", action, err)
	}
	return fmt.Errorf("%s: %w: %s", action, err, msg)
}
