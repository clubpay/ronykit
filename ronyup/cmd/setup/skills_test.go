package setup

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/clubpay/ronykit/ronyup/internal"
)

func TestResolveSkillSelection(t *testing.T) {
	backendDefaults := defaultSkillIDs(KindBackend)
	fullstackDefaults := defaultSkillIDs(KindFullstack)

	if len(backendDefaults) == 0 || len(fullstackDefaults) <= len(backendDefaults) {
		t.Fatalf("expected fullstack defaults to extend backend defaults; backend=%v fullstack=%v", backendDefaults, fullstackDefaults)
	}

	tests := []struct {
		name      string
		requested []string
		kind      string
		want      []string
		wantErr   bool
	}{
		{name: "empty uses backend defaults", requested: nil, kind: KindBackend, want: backendDefaults},
		{name: "empty uses fullstack defaults", requested: nil, kind: KindFullstack, want: fullstackDefaults},
		{name: "none clears", requested: []string{"none"}, kind: KindFullstack, want: nil},
		{name: "all selects everything", requested: []string{"all"}, kind: KindBackend, want: allSkillIDs()},
		{
			name:      "explicit ids in catalog order",
			requested: []string{"go-testing", "go-modern"},
			kind:      KindBackend,
			want:      []string{"go-modern", "go-testing"},
		},
		{
			name:      "comma separated and dedup",
			requested: []string{"go-modern,go-modern", "code-review"},
			kind:      KindBackend,
			want:      []string{"go-modern", "code-review"},
		},
		{
			name:      "default token plus extra",
			requested: []string{"default", "nextjs-modern"},
			kind:      KindBackend,
			want:      append(slices.Clone(backendDefaults), "nextjs-modern"),
		},
		{name: "unknown id errors", requested: []string{"does-not-exist"}, kind: KindBackend, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveSkillSelection(tc.requested, tc.kind)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (got=%v)", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// "default token plus extra" is order-insensitive beyond catalog order;
			// compare as catalog-ordered sets.
			want := filterCatalogOrderSlice(tc.want)
			if !slices.Equal(got, want) {
				t.Fatalf("resolveSkillSelection(%v, %q) = %v, want %v", tc.requested, tc.kind, got, want)
			}
		})
	}
}

func filterCatalogOrderSlice(ids []string) []string {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}

	return filterCatalogOrder(set)
}

func TestCopySkillsWritesSkillFiles(t *testing.T) {
	root := t.TempDir()

	ids := []string{"go-modern", "writing-tests"}
	if err := copySkills(root, ids, nil); err != nil {
		t.Fatalf("copySkills unexpected error: %v", err)
	}

	for _, id := range ids {
		p := filepath.Join(root, ".agents", "skills", id, "SKILL.md")
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected skill file %s: %v", p, err)
		}
	}

	// A skill that was not requested must not be installed.
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "nextjs-modern")); !os.IsNotExist(err) {
		t.Fatalf("did not expect nextjs-modern to be installed (err=%v)", err)
	}
}

func TestCatalogSkillsExistInEmbedFS(t *testing.T) {
	for _, s := range skillCatalog {
		root := t.TempDir()
		if err := copySkills(root, []string{s.ID}, nil); err != nil {
			t.Fatalf("copySkills(%q) error: %v", s.ID, err)
		}

		if _, err := os.Stat(filepath.Join(root, ".agents", "skills", s.ID, "SKILL.md")); err != nil {
			t.Fatalf("catalog skill %q has no embedded SKILL.md: %v", s.ID, err)
		}
	}
}

func TestSkillFrontmatter(t *testing.T) {
	// Bundled ronykit-framework skill (copied with the workspace skeleton) plus
	// every catalog skill. Regression for markdownfmt stripping YAML frontmatter
	// into a markdown heading (see format-markdown.sh).
	paths := []struct {
		id   string
		path string
	}{
		{
			id:   "ronykit-framework",
			path: "skeleton/workspace/.agents/skills/ronykit-framework/SKILL.md",
		},
	}
	for _, s := range skillCatalog {
		paths = append(paths, struct {
			id   string
			path string
		}{
			id:   s.ID,
			path: filepath.ToSlash(filepath.Join(skillsSrcPrefix, s.ID, "SKILL.md")),
		})
	}

	for _, tc := range paths {
		t.Run(tc.id, func(t *testing.T) {
			data, err := internal.Skeleton.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}

			name, desc, ok := parseSkillFrontmatter(string(data))
			if !ok {
				t.Fatalf("%s: missing YAML frontmatter delimited by --- (got leading content %q)",
					tc.path, preview(string(data), 80))
			}

			if name != tc.id {
				t.Fatalf("%s: frontmatter name = %q, want %q", tc.path, name, tc.id)
			}

			if strings.TrimSpace(desc) == "" {
				t.Fatalf("%s: frontmatter description is empty", tc.path)
			}
		})
	}
}

func parseSkillFrontmatter(content string) (name, description string, ok bool) {
	if !strings.HasPrefix(content, "---\n") {
		return "", "", false
	}

	rest := content[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return "", "", false
	}

	fm := rest[:end]
	for line := range strings.SplitSeq(fm, "\n") {
		switch {
		case strings.HasPrefix(line, "name:"):
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		case strings.HasPrefix(line, "description:"):
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			if description == ">-" || description == "|" || description == ">" {
				description = "folded" // non-empty marker for block scalars
			}
		}
	}

	if name == "" || description == "" {
		return name, description, false
	}

	return name, description, true
}

func preview(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > n {
		return s[:n] + "..."
	}

	return s
}
