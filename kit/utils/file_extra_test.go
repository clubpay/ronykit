package utils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/clubpay/ronykit/kit/utils"
)

func TestFileHelpers(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.txt")
	dstPath := filepath.Join(dir, "dst.txt")

	if err := os.WriteFile(srcPath, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := utils.CopyFile(srcPath, dstPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected file content: %q", string(data))
	}

	dstPath2 := filepath.Join(dir, "dst2.txt")
	buf := make([]byte, 2)
	if err := utils.CopyFileWithBuffer(srcPath, dstPath2, buf); err != nil {
		t.Fatal(err)
	}

	data, err = os.ReadFile(dstPath2)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected file content: %q", string(data))
	}

	type sample struct {
		Name string `yaml:"name"`
		Age  int    `yaml:"age"`
	}

	yamlPath := filepath.Join(dir, "sample.yaml")
	if err := utils.WriteYamlFile(yamlPath, sample{Name: "sam", Age: 21}); err != nil {
		t.Fatal(err)
	}

	var decoded sample
	if err := utils.ReadYamlFile(yamlPath, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "sam" || decoded.Age != 21 {
		t.Fatalf("unexpected yaml decode: %+v", decoded)
	}

	if utils.GetExecName() == "" {
		t.Fatal("expected exec name to be non-empty")
	}
	if utils.GetExecDir() == "" {
		t.Fatal("expected exec dir to be non-empty")
	}
}
