package worktree

import "testing"

func TestCreateRejectsEmptyName(t *testing.T) {
	if _, err := Create("/repo", CreateOptions{}); err != ErrInvalidName {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}
