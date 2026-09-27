package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Reuse the current sandbox marker without truncating it during daemon reload.
func ensureContainerEnvFile(dir string) (string, error) {
	if dir == "" {
		return "", errors.New("containerenv requires an infra directory")
	}
	path := filepath.Join(dir, ".containerenv")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err == nil {
		if err := f.Close(); err != nil {
			return "", fmt.Errorf("close containerenv: %w", err)
		}
		return path, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf("create containerenv: %w", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("stat containerenv: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("containerenv is not a regular file: %s", path)
	}
	return path, nil
}
