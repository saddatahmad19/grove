package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesRepoRootFromNestedDirectory(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	nested := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Root != repo {
		t.Fatalf("expected root %q, got %q", repo, cfg.Root)
	}
}

func TestLoadFallsBackToWorkingDirectoryWhenNotInRepo(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Root != tmp {
		t.Fatalf("expected fallback root %q, got %q", tmp, cfg.Root)
	}
}
