package testkit

import "testing"

func TestFolderContent(t *testing.T) {
	files := FolderContent("testdata")
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}
