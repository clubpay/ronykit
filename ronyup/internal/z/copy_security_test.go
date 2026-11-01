package z

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestCopyDirRejectsSymlinkDestinationRoot(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	external := t.TempDir()
	dest := filepath.Join(workspace, ".agents", "skills", "go-modern")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, dest); err != nil {
		t.Fatal(err)
	}

	err := CopyDir(CopyDirParams{
		FS: fstest.MapFS{
			"skills/go-modern/SKILL.md": &fstest.MapFile{Data: []byte("new\n")},
		},
		SrcPathPrefix:  "skills/go-modern",
		DestPathPrefix: dest,
		DestRoot:       workspace,
	})
	if err == nil {
		t.Fatal("expected symlink destination to be rejected")
	}
	if _, err := os.Lstat(filepath.Join(external, "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("copy escaped into symlink target: %v", err)
	}
}

func TestCopyFileRenderErrorPreservesExistingFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dest := filepath.Join(root, "config.yaml")
	if err := os.WriteFile(dest, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := CopyFile(CopyFileParams{
		FS: fstest.MapFS{
			"workspace/config.yamltmpl": &fstest.MapFile{Data: []byte("{{ .Missing }}")},
		},
		SrcPath:        "workspace/config.yamltmpl",
		DestPath:       dest,
		DestRoot:       root,
		TemplateSuffix: "tmpl",
		TemplateInput:  struct{}{},
	})
	if err == nil {
		t.Fatal("expected template execution error")
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "old\n" {
		t.Fatalf("failed render changed destination to %q", content)
	}
}

func TestCopyDirMakesShellScriptsExecutable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := CopyDir(CopyDirParams{
		FS: fstest.MapFS{
			"workspace/verify.sh": &fstest.MapFile{Data: []byte("#!/bin/sh\n")},
		},
		SrcPathPrefix:  "workspace",
		DestPathPrefix: root,
	}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(root, "verify.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("shell script mode = %o, want executable", info.Mode().Perm())
	}
}
