package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillSyncAndValidateOverwriteSemantics(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := SkillSyncRequest{StartDir: root, SkillsMode: []string{"go-modern"}}
	if err := SyncSkills(req, DiscardLogger{}); err != nil {
		t.Fatalf("SyncSkills: %v", err)
	}

	report, err := ValidateSkills(req)
	if err != nil {
		t.Fatalf("ValidateSkills: %v", err)
	}
	if !report.UpToDate {
		t.Fatalf("fresh sync should be current: %+v", report)
	}

	skillFile := filepath.Join(root, ".agents", "skills", "go-modern", "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("local changes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, ".claude", "skills", "go-modern")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(link, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(root, ".cursor", "skills", "go-modern")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := SyncSkills(req, DiscardLogger{}); err != nil {
		t.Fatalf("SyncSkills without overwrite: %v", err)
	}

	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "local changes\n" {
		t.Fatal("non-overwrite sync replaced a modified skill")
	}
	if info, err := os.Lstat(link); err != nil || !info.IsDir() {
		t.Fatalf("non-overwrite sync replaced an existing Claude path: info=%v err=%v", info, err)
	}
	if info, err := os.Lstat(legacy); err != nil || !info.IsDir() {
		t.Fatalf("non-overwrite sync removed a legacy copy: info=%v err=%v", info, err)
	}

	report, err = ValidateSkills(req)
	if err != nil {
		t.Fatal(err)
	}
	if report.UpToDate || !hasSkillIssue(report, "modified_file") ||
		!hasSkillIssue(report, "invalid_link") || !hasSkillIssue(report, "legacy_copy") ||
		!hasSkillIssue(report, "invalid_frontmatter") {
		t.Fatalf("expected all drift types in report: %+v", report)
	}

	req.Overwrite = true
	if err := SyncSkills(req, DiscardLogger{}); err != nil {
		t.Fatalf("SyncSkills with overwrite: %v", err)
	}

	report, err = ValidateSkills(req)
	if err != nil {
		t.Fatal(err)
	}
	if !report.UpToDate {
		t.Fatalf("overwrite sync should repair all drift: %+v", report)
	}
}

func TestValidateSkillsRejectsUnsafeSkillRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".agents", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}

	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "SKILL.md"), []byte("external\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, ".agents", "skills", "go-modern")); err != nil {
		t.Fatal(err)
	}

	report, err := ValidateSkills(SkillSyncRequest{StartDir: root, SkillsMode: []string{"go-modern"}})
	if err != nil {
		t.Fatal(err)
	}
	if !hasSkillIssue(report, "unsafe_path") {
		t.Fatalf("expected unsafe_path issue: %+v", report)
	}
}

func hasSkillIssue(report SkillsReport, kind string) bool {
	for _, status := range report.Skills {
		for _, issue := range status.Issues {
			if issue.Kind == kind {
				return true
			}
		}
	}

	return false
}
