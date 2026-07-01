package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectWorkspaceLayout(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// backend-only
	if err := os.MkdirAll(filepath.Join(root, "backend-only", "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, "backend-only", "go.work"), []byte("go 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	layout, err := detectWorkspaceLayout(filepath.Join(root, "backend-only"))
	if err != nil {
		t.Fatalf("detectWorkspaceLayout(backend-only): %v", err)
	}

	if layout.Kind != KindBackend || layout.GoRoot != layout.RepoRoot {
		t.Fatalf("backend layout: %+v", layout)
	}

	// fullstack at repo root
	fsRoot := filepath.Join(root, "fullstack")
	if err := os.MkdirAll(filepath.Join(fsRoot, "backend"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(fsRoot, "backend", "go.work"), []byte("go 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	layout, err = detectWorkspaceLayout(fsRoot)
	if err != nil {
		t.Fatalf("detectWorkspaceLayout(fullstack root): %v", err)
	}

	if layout.Kind != KindFullstack || layout.GoRoot != filepath.Join(fsRoot, "backend") {
		t.Fatalf("fullstack root layout: %+v", layout)
	}

	// fullstack from backend/ cwd
	layout, err = detectWorkspaceLayout(filepath.Join(fsRoot, "backend"))
	if err != nil {
		t.Fatalf("detectWorkspaceLayout(fullstack backend): %v", err)
	}

	if layout.Kind != KindFullstack || layout.RepoRoot != fsRoot {
		t.Fatalf("fullstack backend cwd layout: %+v", layout)
	}

	// frontend-only
	feRoot := filepath.Join(root, "frontend-only")
	if err := os.MkdirAll(filepath.Join(feRoot, "frontend"), 0o755); err != nil {
		t.Fatal(err)
	}

	layout, err = detectWorkspaceLayout(feRoot)
	if err != nil {
		t.Fatalf("detectWorkspaceLayout(frontend): %v", err)
	}

	if layout.Kind != KindFrontend || layout.GoRoot != "" {
		t.Fatalf("frontend layout: %+v", layout)
	}
}

func TestResolveSyncSections(t *testing.T) {
	t.Parallel()

	sections, err := resolveSyncSections([]string{"devops", "agents"}, KindBackend)
	if err != nil {
		t.Fatalf("resolveSyncSections: %v", err)
	}

	if len(sections) != 2 || sections[0] != syncSectionAgents || sections[1] != syncSectionDevops {
		t.Fatalf("unexpected sections: %v", sections)
	}

	_, err = resolveSyncSections([]string{"nope"}, KindBackend)
	if err == nil {
		t.Fatal("expected error for unknown section")
	}
}

func TestResolveSyncSkillsInstalled(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	skillsDir := filepath.Join(root, ".agents", "skills", "go-modern")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ids, err := resolveSyncSkills(root, []string{"installed"}, KindBackend)
	if err != nil {
		t.Fatalf("resolveSyncSkills: %v", err)
	}

	if len(ids) != 1 || ids[0] != "go-modern" {
		t.Fatalf("installed skills: %v", ids)
	}
}

func TestPathAllowed(t *testing.T) {
	t.Parallel()

	allowed := map[string]bool{"devops": true, "AGENTS.mdtmpl": true}

	if !pathAllowed("devops/devbox/Makefile", allowed) {
		t.Fatal("expected devops child path")
	}

	if pathAllowed("cmd/service/main.go", allowed) {
		t.Fatal("did not expect cmd path")
	}
}
