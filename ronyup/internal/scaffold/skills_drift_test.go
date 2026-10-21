package scaffold

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestRonykitFrameworkSkillCopiesInSync guards against drift between the two
// copies of the ronykit-framework skill: the one agents use in this monorepo
// (.agents/skills/ronykit-framework) and the one ronyup ships into scaffolded
// workspaces (internal/skeleton/workspace/.agents/skills/ronykit-framework).
//
// Every file in the shipped (skeleton) copy must exist in the monorepo copy
// with identical content. Files that exist only in the monorepo copy (e.g.
// README.md) are tolerated.
//
// The test is skipped when the monorepo root (go.work) is not reachable, e.g.
// when the module is checked out standalone.
func TestRonykitFrameworkSkillCopiesInSync(t *testing.T) {
	t.Parallel()

	repoRoot := findRepoRoot(t)
	if repoRoot == "" {
		t.Skip("monorepo root (go.work) not found; skipping drift check")
	}

	monorepoSkill := filepath.Join(repoRoot, ".agents", "skills", "ronykit-framework")
	shippedSkill := filepath.Join(
		repoRoot, "ronyup", "internal", "skeleton", "workspace",
		".agents", "skills", "ronykit-framework",
	)

	if !isDir(shippedSkill) {
		t.Fatalf("shipped skill not found at %s", shippedSkill)
	}

	if !isDir(monorepoSkill) {
		t.Fatalf("monorepo skill not found at %s", monorepoSkill)
	}

	err := filepath.WalkDir(shippedSkill, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(shippedSkill, path)
		if err != nil {
			return err
		}

		shipped, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		monorepoPath := filepath.Join(monorepoSkill, rel)

		monorepo, err := os.ReadFile(monorepoPath)
		if err != nil {
			t.Errorf("monorepo copy missing %s: %v", rel, err)

			return nil
		}

		if !bytes.Equal(shipped, monorepo) {
			t.Errorf(
				"skill file drift: %s differs between monorepo and shipped copies — "+
					"edit both .agents/skills/ronykit-framework/%[1]s and ronyup/internal/skeleton/.../%[1]s together",
				filepath.ToSlash(rel),
			)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk shipped skill: %v", err)
	}
}

// findRepoRoot walks up from the test working directory looking for a go.work
// file (the monorepo root). Returns "" when not found.
func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	for {
		if fileExists(filepath.Join(dir, "go.work")) {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}

		dir = parent
	}
}
