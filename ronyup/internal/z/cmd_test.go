package z

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunCmd_ReturnsErrorOnFailure(t *testing.T) {
	t.Parallel()

	err := RunCmd(context.Background(), RunCmdParams{}, "false")
	if err == nil {
		t.Fatal("expected error from false command")
	}
}

func TestRunCmd_Succeeds(t *testing.T) {
	t.Parallel()

	name := "true"
	if runtime.GOOS == "windows" {
		t.Skip("true binary not assumed on windows")
	}

	if err := RunCmd(context.Background(), RunCmdParams{}, name); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCmd_MergesENVWithProcessEnviron(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("shell quoting differs on windows")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "checkenv.sh")
	content := "#!/bin/sh\ntest -n \"$PATH\" && test \"$RONYUP_TEST_ENV\" = \"1\"\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := RunCmd(context.Background(), RunCmdParams{
		ENV: map[string]string{"RONYUP_TEST_ENV": "1"},
	}, script)
	if err != nil {
		t.Fatalf("expected merged env to include PATH and RONYUP_TEST_ENV: %v", err)
	}
}
