package rkitlint_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clubpay/ronykit/ronyup/internal/rkitlint"
	"github.com/clubpay/ronykit/ronyup/internal/rkitlint/rkitlinttest"
)

// newModule copies testdata/fix into a fresh module (see rkitlinttest.NewModule).
func newModule(t *testing.T, rkit bool) string {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join("testdata", "fix"))
	if err != nil {
		t.Fatal(err)
	}

	files := make(map[string]string)

	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || (!rkit && name == "alias.go") {
			continue
		}

		src, err := os.ReadFile(filepath.Join("testdata", "fix", name))
		if err != nil {
			t.Fatal(err)
		}

		files[name] = string(src)
	}

	return rkitlinttest.NewModule(t, rkit, files)
}

func TestRunFix(t *testing.T) {
	dir := newModule(t, true)

	res, err := rkitlint.Run(rkitlint.Options{Dir: dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := len(res.Findings); got != 20 {
		t.Fatalf("got %d findings, want 20:\n%v", got, res.Findings)
	}

	res, err = rkitlint.Run(rkitlint.Options{Dir: dir, Fix: true})
	if err != nil {
		t.Fatalf("Run fix: %v", err)
	}

	// policy.go's counts() needs two rounds: the loop becomes rkit.Map, then the strconv.Itoa
	// inside the generated closure becomes rkit.IntToStr.
	if len(res.Fixed) != 19 || len(res.Findings) != 1 {
		t.Fatalf("fixed %d, left %d; want 19 fixed and the commented one left:\n%v", len(res.Fixed),
			len(res.Findings), res.Findings)
	}

	if res.Findings[0].Fixable {
		t.Errorf("finding with a comment in range must not offer a fix: %v", res.Findings[0])
	}

	for _, name := range []string{"must.go", "maps.go", "alias.go", "noimports.go", "policy.go"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}

		want, err := os.ReadFile(filepath.Join("testdata", "fix", name+".golden"))
		if err != nil {
			t.Fatal(err)
		}

		if string(got) != string(want) {
			t.Errorf("%s after fix:\n%s\nwant:\n%s", name, got, want)
		}
	}

	build := exec.Command("go", "build", "./...")
	build.Dir = dir

	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("fixed module does not build: %v\n%s", err, out)
	}

	res, err = rkitlint.Run(rkitlint.Options{Dir: dir})
	if err != nil {
		t.Fatalf("Run after fix: %v", err)
	}

	if len(res.Findings) != 1 {
		t.Errorf("after fix got %d findings, want 1: %v", len(res.Findings), res.Findings)
	}
}

func TestRunSkipsModulesWithoutRkit(t *testing.T) {
	dir := newModule(t, false)

	res, err := rkitlint.Run(rkitlint.Options{Dir: dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(res.Findings) != 0 {
		t.Errorf("module without an rkit requirement must be skipped, got %v", res.Findings)
	}

	res, err = rkitlint.Run(rkitlint.Options{Dir: dir, AllModules: true})
	if err != nil {
		t.Fatalf("Run AllModules: %v", err)
	}

	if len(res.Findings) == 0 {
		t.Error("AllModules must check modules without an rkit requirement")
	}

	// Offline, the second fix round cannot load the module until go mod tidy adds rkit; Run must
	// keep the first round's fixes instead of failing.
	t.Setenv("GOPROXY", "off")

	res, err = rkitlint.Run(rkitlint.Options{Dir: dir, AllModules: true, Fix: true})
	if err != nil {
		t.Fatalf("Run AllModules fix: %v", err)
	}

	if len(res.Fixed) == 0 || len(res.NeedsTidy) != 1 || !sameDir(res.NeedsTidy[0], dir) {
		t.Errorf("fixing a module without an rkit requirement must ask for go mod tidy in %s, got %v",
			dir, res.NeedsTidy)
	}
}

func sameDir(a, b string) bool {
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)

	return errA == nil && errB == nil && ra == rb
}

func TestRunLoadError(t *testing.T) {
	dir := newModule(t, true)
	if err := os.WriteFile(filepath.Join(dir, "broken.go"),
		[]byte("package fixmod\n\nfunc broken() { undefined() }\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := rkitlint.Run(rkitlint.Options{Dir: dir})

	var loadErr *rkitlint.LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("want *LoadError, got %v", err)
	}
}
