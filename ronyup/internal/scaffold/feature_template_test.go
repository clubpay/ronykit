package scaffold

import (
	"io/fs"
	"strings"
	"testing"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/clubpay/ronykit/ronyup/internal"
)

func TestServiceTemplatesParse(t *testing.T) {
	t.Parallel()

	input := TemplateInput{
		RepositoryPath: "github.com/example/app",
		PackagePath:    "feature/billing",
		PackageName:    "billing",
		RonyKitPath:    RonyKitModulePath,
	}

	err := fs.WalkDir(internal.Skeleton, "skeleton/service", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() || !strings.HasSuffix(path, "tmpl") {
			return nil
		}

		raw, err := fs.ReadFile(internal.Skeleton, path)
		if err != nil {
			return err
		}

		tmpl, err := template.New(path).Funcs(sprig.FuncMap()).Parse(string(raw))
		if err != nil {
			t.Errorf("parse %s: %v", path, err)

			return nil
		}

		var buf strings.Builder
		if err := tmpl.Execute(&buf, input); err != nil {
			t.Errorf("execute %s: %v", path, err)
		}

		if strings.Contains(buf.String(), "{{") {
			t.Errorf("%s left unexpanded template braces", path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk service skeleton: %v", err)
	}
}

func TestValidFeatureTemplate(t *testing.T) {
	t.Parallel()

	if got := FeatureTemplates; len(got) != 1 || got[0] != "service" {
		t.Fatalf("FeatureTemplates = %v, want [service]", got)
	}

	if !validFeatureTemplate("service") {
		t.Fatal("service must be a valid feature template")
	}

	for _, name := range []string{"job", "gateway", "worker", ""} {
		if validFeatureTemplate(name) {
			t.Fatalf("%q must not be a valid feature template", name)
		}
	}
}
