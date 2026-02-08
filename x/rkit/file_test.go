package rkit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecPaths(t *testing.T) {
	if got := GetExecName(); got != filepath.Base(os.Args[0]) {
		t.Fatalf("GetExecName = %q, want %q", got, filepath.Base(os.Args[0]))
	}

	wantDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		t.Fatalf("filepath.Abs = %v", err)
	}
	if got := GetExecDir(); got != wantDir {
		t.Fatalf("GetExecDir = %q, want %q", got, wantDir)
	}
}

func TestGetCurrentDir(t *testing.T) {
	want, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd = %v", err)
	}
	if got := GetCurrentDir(); got != want {
		t.Fatalf("GetCurrentDir = %q, want %q", got, want)
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile src = %v", err)
	}
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile = %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile dst = %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("CopyFile content = %q, want %q", string(got), "hello")
	}
}

func TestCopyFileWithBuffer(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	if err := os.WriteFile(src, []byte("world"), 0o644); err != nil {
		t.Fatalf("WriteFile src = %v", err)
	}

	buf := make([]byte, 2)
	if err := CopyFileWithBuffer(src, dst, buf); err != nil {
		t.Fatalf("CopyFileWithBuffer = %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile dst = %v", err)
	}
	if string(got) != "world" {
		t.Fatalf("CopyFileWithBuffer content = %q, want %q", string(got), "world")
	}
}

func TestYamlHelpers(t *testing.T) {
	type payload struct {
		Name string `yaml:"name"`
		Age  int    `yaml:"age"`
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "data.yaml")
	want := payload{Name: "Zed", Age: 33}

	if err := WriteYamlFile(path, want); err != nil {
		t.Fatalf("WriteYamlFile = %v", err)
	}

	var got payload
	if err := ReadYamlFile(path, &got); err != nil {
		t.Fatalf("ReadYamlFile = %v", err)
	}

	if got != want {
		t.Fatalf("ReadYamlFile = %+v, want %+v", got, want)
	}
}
