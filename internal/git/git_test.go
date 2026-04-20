package git

import "testing"

func TestParseWorktreePorcelain(t *testing.T) {
	out := []byte("worktree /repo\nHEAD abc123\nbranch refs/heads/main\nbare false\n\nworktree /repo-feat\nHEAD def456\nbranch refs/heads/feat\n\n")
	items := parseWorktreePorcelain(out)
	if len(items) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(items))
	}
	if items[0].Path != "/repo" || items[0].Branch != "main" || items[0].Head != "abc123" || items[0].Bare {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[1].Path != "/repo-feat" || items[1].Branch != "feat" || items[1].Head != "def456" {
		t.Fatalf("unexpected second item: %+v", items[1])
	}
}

func TestCommandErrorIncludesOutput(t *testing.T) {
	err := commandError("create worktree", assertErr{}, []byte("branch exists"))
	if got := err.Error(); got == "" || got == "create worktree: assert err" {
		t.Fatalf("unexpected error: %v", err)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "assert err" }
