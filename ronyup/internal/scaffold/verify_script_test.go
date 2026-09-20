package scaffold

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBackendVerifyScriptSyntax(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not on PATH")
	}

	script := filepath.Join("..", "skeleton", "backend", "verify.sh")
	out, err := exec.Command("bash", "-n", script).CombinedOutput()
	if err != nil {
		t.Fatalf("bash -n %s: %v\n%s", script, err, out)
	}
}
