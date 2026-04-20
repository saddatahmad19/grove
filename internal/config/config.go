package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Root string
}

func Load() (Config, error) {
	root := os.Getenv("GROVE_ROOT")
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return Config{}, err
		}
		root, err = discoverRepoRoot(cwd)
		if err != nil {
			return Config{}, err
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Config{}, err
	}
	return Config{Root: abs}, nil
}

func discoverRepoRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(current, ".git")); err == nil && info.IsDir() {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return start, nil
		}
		current = parent
	}
}
