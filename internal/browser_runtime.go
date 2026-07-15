package internal

import (
	"fmt"
	"os"
	"path/filepath"
)

func ensureChromiumRuntimeDirs() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve Chromium home directory: %w", err)
	}
	if home == "" {
		return fmt.Errorf("resolve Chromium home directory: HOME is empty")
	}
	for _, dir := range []string{
		filepath.Join(home, ".config", "chromium", "Crash Reports"),
		filepath.Join(home, ".cache", "chromium"),
	} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create Chromium runtime directory %s: %w", dir, err)
		}
	}
	return nil
}
