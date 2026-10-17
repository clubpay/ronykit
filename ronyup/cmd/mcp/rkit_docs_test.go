package mcp

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestRkitDocsCoverExportedAPI fails when an exported x/rkit (or rkit/reflector) identifier is
// missing from the MCP resource packages/rkit.md or from x/rkit/README.md, so the docs agents read
// cannot fall behind the code. Runs only inside the monorepo.
func TestRkitDocsCoverExportedAPI(t *testing.T) {
	root := monorepoRoot(t)
	rkitDir := filepath.Join(root, "x", "rkit")

	names := append(exportedNames(t, rkitDir), exportedNames(t, filepath.Join(rkitDir, "reflector"))...)
	if len(names) < 50 {
		t.Fatalf("found only %d exported rkit identifiers; is %s the rkit package?", len(names), rkitDir)
	}

	docs := []string{
		filepath.Join("knowledge", "resources", "packages", "rkit.md"),
		filepath.Join(rkitDir, "README.md"),
	}

	for _, doc := range docs {
		raw, err := os.ReadFile(doc)
		if err != nil {
			t.Fatal(err)
		}

		for _, name := range names {
			// Docs spell identifiers as `Name`, rkit.Name or `A`/`B` lists.
			if !regexp.MustCompile("[`./]" + regexp.QuoteMeta(name) + `\b`).Match(raw) {
				t.Errorf("%s does not mention %s", doc, name)
			}
		}
	}
}

// exportedNames returns the exported top-level funcs, types, vars and consts of the package in dir.
func exportedNames(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	var names []string

	fset := token.NewFileSet()

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}

		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}

		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.IsExported() {
					names = append(names, d.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							names = append(names, s.Name.Name)
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.IsExported() {
								names = append(names, n.Name)
							}
						}
					}
				}
			}
		}
	}

	return names
}

func monorepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("monorepo root (go.work) not found")
		}

		dir = parent
	}
}
