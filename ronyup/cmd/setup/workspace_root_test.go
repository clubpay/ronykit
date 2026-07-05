package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveGoWorkspace_BackendRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := resolveGoWorkspace(root)
	if err != nil {
		t.Fatalf("resolveGoWorkspace() unexpected error: %v", err)
	}

	if got != root {
		t.Fatalf("got %q, want %q", got, root)
	}
}

func TestResolveGoWorkspace_FullstackRepoRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	backend := filepath.Join(root, backendDir)
	if err := os.MkdirAll(backend, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := os.WriteFile(filepath.Join(backend, "go.work"), []byte("go 1.25\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := resolveGoWorkspace(root)
	if err != nil {
		t.Fatalf("resolveGoWorkspace() unexpected error: %v", err)
	}

	if got != backend {
		t.Fatalf("got %q, want %q", got, backend)
	}
}

func TestResolveGoWorkspace_FullstackBackendDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	backend := filepath.Join(root, backendDir)
	if err := os.MkdirAll(backend, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := os.WriteFile(filepath.Join(backend, "go.work"), []byte("go 1.25\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := resolveGoWorkspace(backend)
	if err != nil {
		t.Fatalf("resolveGoWorkspace() unexpected error: %v", err)
	}

	if got != backend {
		t.Fatalf("got %q, want %q", got, backend)
	}
}
