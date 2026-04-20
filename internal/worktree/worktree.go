package worktree

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/saddatahmad19/grove/internal/git"
)

type Worktree struct {
	Path   string
	Branch string
	Head   string
	Name   string
}

type CreateOptions struct {
	Name   string
	Branch string
	Start  string
}

var ErrInvalidName = errors.New("worktree name is required")

func LoadAll(root string) ([]Worktree, error) {
	items, err := git.ListWorktrees(root)
	if err != nil {
		return nil, err
	}
	out := make([]Worktree, 0, len(items))
	for _, item := range items {
		out = append(out, Worktree{
			Path:   item.Path,
			Branch: item.Branch,
			Head:   item.Head,
			Name:   filepath.Base(item.Path),
		})
	}
	return out, nil
}

func Create(root string, opts CreateOptions) (Worktree, error) {
	if strings.TrimSpace(opts.Name) == "" {
		return Worktree{}, ErrInvalidName
	}
	path := filepath.Join(filepath.Dir(root), opts.Name)
	_, err := git.CreateWorktree(root, git.WorktreeCreateOptions{
		Path:   path,
		Branch: opts.Branch,
		Start:  opts.Start,
	})
	if err != nil {
		return Worktree{}, err
	}
	wt := Worktree{Path: path, Name: opts.Name, Branch: opts.Branch, Head: opts.Start}
	if wt.Head == "" {
		wt.Head = "HEAD"
	}
	return wt, nil
}
