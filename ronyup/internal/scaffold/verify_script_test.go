package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyScriptsSyntax(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not on PATH")
	}

	scripts := []string{
		filepath.Join("..", "skeleton", "backend", "verify.sh"),
		filepath.Join("..", "skeleton", "frontend", "verify.sh"),
		filepath.Join("..", "skeleton", "workspace", ".cursor", "hooks", "backend-verify.sh"),
		filepath.Join("..", "skeleton", "workspace", ".cursor", "hooks", "frontend-verify.sh"),
		filepath.Join("..", "skeleton", "workspace", ".claude", "hooks", "backend-verify.sh"),
		filepath.Join("..", "skeleton", "workspace", ".claude", "hooks", "backend-lint.sh"),
		filepath.Join("..", "skeleton", "workspace", ".claude", "hooks", "frontend-verify.sh"),
	}

	for _, script := range scripts {
		t.Run(script, func(t *testing.T) {
			t.Parallel()

			out, err := exec.Command("bash", "-n", script).CombinedOutput()
			if err != nil {
				t.Fatalf("bash -n %s: %v\n%s", script, err, out)
			}
		})
	}
}

func TestFrontendVerifyScript_UIPolishCheck(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not on PATH")
	}

	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}

	tmp := t.TempDir()
	frontendDir := filepath.Join(tmp, "frontend")
	designDir := filepath.Join(tmp, "docs", "design")

	if err := os.MkdirAll(filepath.Join(frontendDir, "node_modules"), 0o755); err != nil {
		t.Fatalf("MkdirAll(frontend/node_modules): %v", err)
	}

	if err := os.MkdirAll(designDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(docs/design): %v", err)
	}

	srcVerify, err := os.ReadFile(filepath.Join("..", "skeleton", "frontend", "verify.sh"))
	if err != nil {
		t.Fatalf("ReadFile(frontend/verify.sh): %v", err)
	}

	verifyPath := filepath.Join(frontendDir, "verify.sh")
	if err := os.WriteFile(verifyPath, srcVerify, 0o755); err != nil {
		t.Fatalf("WriteFile(verify.sh): %v", err)
	}

	if err := os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte(`{"name":"web","scripts":{}}`), 0o644); err != nil {
		t.Fatalf("WriteFile(package.json): %v", err)
	}

	if err := os.WriteFile(filepath.Join(designDir, "web-frontend-design.md"), []byte("---\nstatus: approved\n---\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(design doc): %v", err)
	}

	compDir := filepath.Join(frontendDir, "src", "components")
	if err := os.MkdirAll(compDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(components): %v", err)
	}

	badComponent := `export function Card() {
  return <div className="min-h-screen w-screen transition-all scale-0"> Bad </div>;
}
`
	if err := os.WriteFile(filepath.Join(compDir, "Card.tsx"), []byte(badComponent), 0o644); err != nil {
		t.Fatalf("WriteFile(Card.tsx): %v", err)
	}

	out, err := exec.Command("bash", verifyPath).CombinedOutput()
	if err == nil {
		t.Fatalf("expected verify.sh to fail on UI polish violations, got success:\n%s", out)
	}

	for _, want := range []string{"transition-all", "100dvh", "w-full", "scale(0)"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("expected output to mention %q, got:\n%s", want, out)
		}
	}

	goodComponent := `export function Card() {
  return <div className="min-h-dvh w-full transition-[transform,opacity] scale-95"> Good </div>;
}
`
	if err := os.WriteFile(filepath.Join(compDir, "Card.tsx"), []byte(goodComponent), 0o644); err != nil {
		t.Fatalf("WriteFile(Card.tsx): %v", err)
	}

	out, err = exec.Command("bash", verifyPath).CombinedOutput()
	if err != nil {
		t.Fatalf("expected verify.sh to pass once fixed, got err=%v:\n%s", err, out)
	}
}
