package main

import (
	"os"
	"path/filepath"
)

// dataDir returns the per-user state directory (~/.vlessbar), creating it.
func dataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".vlessbar")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}
