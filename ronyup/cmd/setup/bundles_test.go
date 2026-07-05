package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFeatureImports(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "features.go")
	content := `package main

import (
	_ "github.com/example/app/feature/auth"
	_ "github.com/example/app/feature/user"
)
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	imports, err := parseFeatureImports(path)
	if err != nil {
		t.Fatalf("parseFeatureImports() unexpected error: %v", err)
	}

	want := []string{
		"github.com/example/app/feature/auth",
		"github.com/example/app/feature/user",
	}
	if len(imports) != len(want) {
		t.Fatalf("got %d imports, want %d: %v", len(imports), len(want), imports)
	}

	for i, imp := range imports {
		if imp != want[i] {
			t.Fatalf("imports[%d] = %q, want %q", i, imp, want[i])
		}
	}
}

func TestFilterImportsForBundle(t *testing.T) {
	t.Parallel()

	all := []string{
		"github.com/example/app/feature/auth",
		"github.com/example/app/feature/user",
	}

	t.Run("wildcard", func(t *testing.T) {
		t.Parallel()

		got, err := filterImportsForBundle(all, "github.com/example/app", BundleSpec{
			Services: []string{wildcardService},
		})
		if err != nil {
			t.Fatalf("filterImportsForBundle() unexpected error: %v", err)
		}

		if len(got) != len(all) {
			t.Fatalf("got %d imports, want %d", len(got), len(all))
		}
	})

	t.Run("subset", func(t *testing.T) {
		t.Parallel()

		got, err := filterImportsForBundle(all, "github.com/example/app", BundleSpec{
			Services: []string{"feature/auth"},
		})
		if err != nil {
			t.Fatalf("filterImportsForBundle() unexpected error: %v", err)
		}

		if len(got) != 1 || got[0] != all[0] {
			t.Fatalf("got %v, want [%q]", got, all[0])
		}
	})

	t.Run("missing service", func(t *testing.T) {
		t.Parallel()

		_, err := filterImportsForBundle(all, "github.com/example/app", BundleSpec{
			Services: []string{"feature/billing"},
		})
		if err == nil {
			t.Fatal("expected error for missing service")
		}
	})
}

func TestRenderFeaturesGo(t *testing.T) {
	t.Parallel()

	content := renderFeaturesGo([]string{
		"github.com/example/app/feature/auth",
	})

	if !strings.Contains(content, "package main") || !strings.Contains(content, `_ "github.com/example/app/feature/auth"`) {
		t.Fatalf("unexpected content:\n%s", content)
	}
}

func TestLoadBundlesConfig_DefaultWhenMissing(t *testing.T) {
	t.Parallel()

	cfg, err := loadBundlesConfig(t.TempDir())
	if err != nil {
		t.Fatalf("loadBundlesConfig() unexpected error: %v", err)
	}

	spec, ok := cfg.Bundles[defaultBundleName]
	if !ok {
		t.Fatalf("expected default bundle %q", defaultBundleName)
	}

	if len(spec.Services) != 1 || spec.Services[0] != wildcardService {
		t.Fatalf("unexpected services: %v", spec.Services)
	}
}
