// Package rkitlinttest builds throwaway Go modules for rkitlint tests.
package rkitlinttest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const rkitPath = "github.com/clubpay/ronykit/x/rkit"

// NewModule writes files into a fresh module directory and returns it. With rkit set, the module
// requires x/rkit, replaced by this repository's copy so fixed code can be compiled. It sets
// GOWORK=off and GOFLAGS=-mod=mod for the rest of the test.
func NewModule(tb testing.TB, rkit bool, files map[string]string) string {
	tb.Helper()

	tb.Setenv("GOWORK", "off")
	tb.Setenv("GOFLAGS", "-mod=mod")

	dir := tb.TempDir()
	gomod := "module example.com/lintmod\n\ngo 1.26\n"

	if rkit {
		rkitDir := RkitDir(tb)
		gomod += "\nrequire " + rkitPath + " v0.0.0\n\nreplace " + rkitPath + " => " + rkitDir + "\n"

		sum, err := os.ReadFile(filepath.Join(rkitDir, "go.sum"))
		if err != nil {
			tb.Fatal(err)
		}

		write(tb, filepath.Join(dir, "go.sum"), sum)
	}

	write(tb, filepath.Join(dir, "go.mod"), []byte(gomod))

	for name, content := range files {
		write(tb, filepath.Join(dir, name), []byte(content))
	}

	return dir
}

// RkitDir returns the x/rkit directory of this repository.
func RkitDir(tb testing.TB) string {
	tb.Helper()

	_, self, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("cannot locate rkitlinttest source")
	}

	// self is <repo>/ronyup/internal/rkitlint/rkitlinttest/module.go.
	return filepath.Join(filepath.Dir(self), "..", "..", "..", "..", "x", "rkit")
}

func write(tb testing.TB, path string, data []byte) {
	tb.Helper()

	if err := os.WriteFile(path, data, 0o600); err != nil {
		tb.Fatal(err)
	}
}
