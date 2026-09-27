package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureContainerEnvFile(t *testing.T) {
	dir := t.TempDir()
	path, err := ensureContainerEnvFile(dir)
	if err != nil || path != filepath.Join(dir, ".containerenv") {
		t.Fatalf("create: %q, %v", path, err)
	}
	if err := os.WriteFile(path, []byte("preserve marker"), 0o600); err != nil {
		t.Fatal(err)
	}
	again, err := ensureContainerEnvFile(dir)
	if err != nil || again != path {
		t.Fatalf("reload: %q, %v", again, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "preserve marker" {
		t.Fatalf("marker changed: %q, %v", data, err)
	}
}

func TestEnsureContainerEnvFileFailure(t *testing.T) {
	for _, dir := range []string{"", filepath.Join(t.TempDir(), "missing")} {
		if path, err := ensureContainerEnvFile(dir); err == nil || path != "" {
			t.Fatalf("invalid directory accepted: %q, %v", path, err)
		}
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".containerenv"), 0o700); err != nil {
		t.Fatal(err)
	}
	if path, err := ensureContainerEnvFile(dir); err == nil || path != "" {
		t.Fatalf("directory marker accepted: %q, %v", path, err)
	}
}
