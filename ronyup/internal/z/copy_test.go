package z

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestCopyFile_NonTemplateFileIsCopiedVerbatim(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	destPath := filepath.Join(root, "Makefile")
	source := fstest.MapFS{
		"workspace/Makefile": &fstest.MapFile{
			Data: []byte("MODULE_DIRS := $(shell go list -f '{{.Dir}}' -m all)\n"),
		},
	}

	err := CopyFile(CopyFileParams{
		FS:             source,
		SrcPath:        "workspace/Makefile",
		DestPath:       destPath,
		TemplateSuffix: "tmpl",
		TemplateInput: struct {
			RepositoryPath string
		}{
			RepositoryPath: "github.com/example/repo",
		},
	})
	if err != nil {
		t.Fatalf("CopyFile() unexpected error: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("ReadFile() unexpected error: %v", err)
	}

	want := "MODULE_DIRS := $(shell go list -f '{{.Dir}}' -m all)\n"
	if string(got) != want {
		t.Fatalf("copied file mismatch\nwant: %q\ngot:  %q", want, string(got))
	}
}

func TestCopyFile_TemplateFileIsRendered(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	destPath := filepath.Join(root, "config.yaml")
	source := fstest.MapFS{
		"workspace/config.yamltmpl": &fstest.MapFile{
			Data: []byte("repo: {{ .RepositoryPath }}\n"),
		},
	}

	err := CopyFile(CopyFileParams{
		FS:             source,
		SrcPath:        "workspace/config.yamltmpl",
		DestPath:       destPath,
		TemplateSuffix: "tmpl",
		TemplateInput: struct {
			RepositoryPath string
		}{
			RepositoryPath: "github.com/example/repo",
		},
	})
	if err != nil {
		t.Fatalf("CopyFile() unexpected error: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("ReadFile() unexpected error: %v", err)
	}

	want := "repo: github.com/example/repo\n"
	if string(got) != want {
		t.Fatalf("rendered file mismatch\nwant: %q\ngot:  %q", want, string(got))
	}
}

func TestCopyFile_TemplateInputNilCopiesTemplateFileVerbatim(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	destPath := filepath.Join(root, "raw.tmpl")
	source := fstest.MapFS{
		"workspace/raw.tmpl": &fstest.MapFile{
			Data: []byte("{{ .RepositoryPath }}\n"),
		},
	}

	err := CopyFile(CopyFileParams{
		FS:             source,
		SrcPath:        "workspace/raw.tmpl",
		DestPath:       destPath,
		TemplateSuffix: "tmpl",
		TemplateInput:  nil,
	})
	if err != nil {
		t.Fatalf("CopyFile() unexpected error: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("ReadFile() unexpected error: %v", err)
	}

	want := "{{ .RepositoryPath }}\n"
	if string(got) != want {
		t.Fatalf("copied file mismatch\nwant: %q\ngot:  %q", want, string(got))
	}
}

func TestCopyDir_RendersOnlyTemplateSuffixFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sourceRoot := filepath.Join(root, "source")
	destRoot := filepath.Join(root, "dest")

	err := os.MkdirAll(filepath.Join(sourceRoot, "workspace"), 0o755)
	if err != nil {
		t.Fatalf("MkdirAll() unexpected error: %v", err)
	}

	err = os.WriteFile(
		filepath.Join(sourceRoot, "workspace", "Makefile"),
		[]byte("MODULE_DIRS := $(shell go list -f '{{.Dir}}' -m all)\n"),
		0o644,
	)
	if err != nil {
		t.Fatalf("WriteFile(Makefile) unexpected error: %v", err)
	}

	err = os.WriteFile(
		filepath.Join(sourceRoot, "workspace", "go.modtmpl"),
		[]byte("module {{ .RepositoryPath }}\n"),
		0o644,
	)
	if err != nil {
		t.Fatalf("WriteFile(go.modtmpl) unexpected error: %v", err)
	}

	err = CopyDir(CopyDirParams{
		FS:             os.DirFS(sourceRoot),
		SrcPathPrefix:  "workspace",
		DestPathPrefix: destRoot,
		TemplateInput: struct {
			RepositoryPath string
		}{
			RepositoryPath: "github.com/example/repo",
		},
	})
	if err != nil {
		t.Fatalf("CopyDir() unexpected error: %v", err)
	}

	makefileData, err := os.ReadFile(filepath.Join(destRoot, "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(Makefile) unexpected error: %v", err)
	}

	if string(makefileData) != "MODULE_DIRS := $(shell go list -f '{{.Dir}}' -m all)\n" {
		t.Fatalf("Makefile should be copied without template rendering, got: %q", string(makefileData))
	}

	goModData, err := os.ReadFile(filepath.Join(destRoot, "go.mod"))
	if err != nil {
		t.Fatalf("ReadFile(go.mod) unexpected error: %v", err)
	}

	if string(goModData) != "module github.com/example/repo\n" {
		t.Fatalf("go.mod template rendering mismatch, got: %q", string(goModData))
	}
}
