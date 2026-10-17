package scaffold

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestMonorepoPathRulesMirrored pins the monorepo's Claude Code rules (.claude/rules/*.md) to the
// Cursor rules they mirror (.cursor/rules/*.mdc): same body, and `paths:` equal to `globs:`.
// The Claude copies quote their globs; an unparseable frontmatter would make Claude Code load the
// rule for every file.
func TestMonorepoPathRulesMirrored(t *testing.T) {
	t.Parallel()

	repoRoot := findRepoRoot(t)
	if repoRoot == "" {
		t.Skip("monorepo root (go.work) not found; skipping rules drift check")
	}

	cursorRules, err := filepath.Glob(filepath.Join(repoRoot, ".cursor", "rules", "*.mdc"))
	if err != nil || len(cursorRules) == 0 {
		t.Fatalf("no Cursor rules found (err=%v)", err)
	}

	for _, cursorPath := range cursorRules {
		name := strings.TrimSuffix(filepath.Base(cursorPath), ".mdc")
		claudePath := filepath.Join(repoRoot, ".claude", "rules", name+".md")

		cursorFM, cursorBody := splitFrontmatter(t, cursorPath)
		claudeFM, claudeBody := splitFrontmatter(t, claudePath)

		if cursorBody != claudeBody {
			t.Errorf("%s body differs from %s; copy the Cursor rule body over", claudePath, cursorPath)
		}

		var claude struct {
			Paths []string `yaml:"paths"`
		}
		if err := yaml.Unmarshal([]byte(claudeFM), &claude); err != nil {
			t.Errorf("%s frontmatter is not valid YAML: %v", claudePath, err)

			continue
		}

		var globs []string

		for line := range strings.SplitSeq(cursorFM, "\n") {
			if v, ok := strings.CutPrefix(line, "globs:"); ok {
				for g := range strings.SplitSeq(v, ",") {
					globs = append(globs, strings.TrimSpace(g))
				}
			}
		}

		slices.Sort(globs)
		slices.Sort(claude.Paths)

		if !slices.Equal(globs, claude.Paths) {
			t.Errorf("%s paths %v differ from %s globs %v", claudePath, claude.Paths, cursorPath, globs)
		}
	}
}

func splitFrontmatter(t *testing.T, path string) (frontmatter, body string) {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}

	rest, ok := strings.CutPrefix(string(raw), "---\n")
	if !ok {
		t.Fatalf("%s has no frontmatter", path)
	}

	frontmatter, body, ok = strings.Cut(rest, "\n---\n")
	if !ok {
		t.Fatalf("%s frontmatter is not closed", path)
	}

	return frontmatter, body
}
