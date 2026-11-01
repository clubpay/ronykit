package z

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestRunCmdErrorIncludesCapturedOutput(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("shell quoting differs on windows")
	}

	err := RunCmd(context.Background(), RunCmdParams{}, "sh", "-c", "echo command-detail; exit 1")
	if err == nil || !strings.Contains(err.Error(), "command-detail") {
		t.Fatalf("expected captured command output in error, got %v", err)
	}
}
