package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFrontmatterStatus(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "approved", content: "---\nstatus: approved\n---\n# SRS", want: "approved"},
		{name: "draft", content: "---\nfeature: billing\nstatus: draft\n---\nbody", want: "draft"},
		{name: "quoted", content: "---\nstatus: \"approved\"\n---", want: "approved"},
		{name: "inline comment", content: "---\nstatus: draft # set later\n---", want: "draft"},
		{name: "leading blank lines", content: "\n\n---\nstatus: approved\n---", want: "approved"},
		{name: "bom prefix", content: "\ufeff---\nstatus: approved\n---", want: "approved"},
		{name: "no frontmatter", content: "# SRS\nstatus: approved", want: ""},
		{name: "no status key", content: "---\nfeature: billing\n---", want: ""},
		{name: "status after close", content: "---\nfeature: x\n---\nstatus: approved", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := frontmatterStatus(tc.content); got != tc.want {
				t.Fatalf("frontmatterStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

func writeDesignDoc(t *testing.T, ws, feature, suffix, status string) {
	t.Helper()

	dir := filepath.Join(ws, designDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	content := "# doc"
	if status != "" {
		content = "---\nfeature: " + feature + "\nstatus: " + status + "\n---\n# doc"
	}

	if err := os.WriteFile(filepath.Join(dir, feature+suffix), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestCheckDesignGate(t *testing.T) {
	t.Run("both missing", func(t *testing.T) {
		ws := t.TempDir()
		if got := checkDesignGate(ws, "billing"); len(got) != 2 {
			t.Fatalf("expected 2 problems, got %d: %v", len(got), got)
		}
	})

	t.Run("srs approved, sdd missing", func(t *testing.T) {
		ws := t.TempDir()
		writeDesignDoc(t, ws, "billing", "-srs.md", "approved")
		if got := checkDesignGate(ws, "billing"); len(got) != 1 {
			t.Fatalf("expected 1 problem, got %d: %v", len(got), got)
		}
	})

	t.Run("both draft", func(t *testing.T) {
		ws := t.TempDir()
		writeDesignDoc(t, ws, "billing", "-srs.md", "draft")
		writeDesignDoc(t, ws, "billing", "-sdd.md", "draft")
		if got := checkDesignGate(ws, "billing"); len(got) != 2 {
			t.Fatalf("expected 2 problems, got %d: %v", len(got), got)
		}
	})

	t.Run("no frontmatter status", func(t *testing.T) {
		ws := t.TempDir()
		writeDesignDoc(t, ws, "billing", "-srs.md", "")
		writeDesignDoc(t, ws, "billing", "-sdd.md", "")
		if got := checkDesignGate(ws, "billing"); len(got) != 2 {
			t.Fatalf("expected 2 problems, got %d: %v", len(got), got)
		}
	})

	t.Run("both approved passes", func(t *testing.T) {
		ws := t.TempDir()
		writeDesignDoc(t, ws, "billing", "-srs.md", "approved")
		writeDesignDoc(t, ws, "billing", "-sdd.md", "approved")
		if got := checkDesignGate(ws, "billing"); len(got) != 0 {
			t.Fatalf("expected gate to pass, got problems: %v", got)
		}
	})

	t.Run("approved case-insensitive", func(t *testing.T) {
		ws := t.TempDir()
		writeDesignDoc(t, ws, "billing", "-srs.md", "Approved")
		writeDesignDoc(t, ws, "billing", "-sdd.md", "APPROVED")
		if got := checkDesignGate(ws, "billing"); len(got) != 0 {
			t.Fatalf("expected gate to pass, got problems: %v", got)
		}
	})
}

func TestCheckFrontendDesignGate(t *testing.T) {
	completeApprovedDoc := `---
status: approved
app: web
stack: Next.js 16 App Router + shadcn/ui + Tailwind
updated: 2026-10-18
---

# Web — frontend design

## 1. Context
Operate dashboard for billing ops.

## 2. Aesthetic direction
Precision dark-first editorial ledger.

## 3. Token plan
OKLCH 4-tier text hierarchy and surface steps.

## 4. Layout
240px sidebar + 1440px max-width workspace.

## 5. Component rules
shadcn/ui + Lucide icons, tabular-nums tables.

## 6. Motion
0ms high-frequency actions, cubic-bezier(0.23, 1, 0.32, 1) overlays.

## 7. v1 surfaces
/invoices, /customers.

## 9. Quality floor
Responsive >=360px, Storybook coverage, verify.sh clean.
`

	t.Run("missing file", func(t *testing.T) {
		ws := t.TempDir()
		got := checkFrontendDesignGate(ws, "web")
		if len(got) != 1 {
			t.Fatalf("expected 1 problem, got %d: %v", len(got), got)
		}
	})

	t.Run("draft and missing sections and comments", func(t *testing.T) {
		ws := t.TempDir()
		dir := filepath.Join(ws, designDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		raw := "---\nstatus: draft\n---\n# Web\n## 1. Context\n<!-- fill me -->\n"
		if err := os.WriteFile(filepath.Join(dir, "web-frontend-design.md"), []byte(raw), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		got := checkFrontendDesignGate(ws, "web")
		if len(got) < 3 {
			t.Fatalf("expected multiple problems (draft, missing sections, comments), got: %v", got)
		}
	})

	t.Run("complete and approved passes", func(t *testing.T) {
		ws := t.TempDir()
		dir := filepath.Join(ws, designDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "web-frontend-design.md"), []byte(completeApprovedDoc), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if got := checkFrontendDesignGate(ws, ""); len(got) != 0 {
			t.Fatalf("expected gate to pass, got problems: %v", got)
		}
	})
}
