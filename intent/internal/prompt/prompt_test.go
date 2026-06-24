package prompt

import (
	"strings"
	"testing"
)

func TestSkillCatalog(t *testing.T) {
	catalog := SkillCatalog(SkillCatalogData{
		Title:             SkillCatalogTitle,
		ActivateSkillTool: "activate_skill",
		Skills: []SkillRow{
			{
				Name:        "billing",
				Description: "Handle refunds and billing inquiries.",
				Triggers:    []string{"refund", "chargeback"},
				Examples:    []string{"I want a refund for order 42"},
			},
			{Name: "support", Description: "Answer product questions."},
		},
	})

	if !strings.Contains(catalog, "## Skill Catalog") {
		t.Fatalf("expected Skill Catalog heading, got:\n%s", catalog)
	}
	if !strings.Contains(catalog, "1. **billing** — Handle refunds and billing inquiries.") {
		t.Fatalf("expected billing entry, got:\n%s", catalog)
	}
	if !strings.Contains(catalog, "Use when: refund, chargeback") {
		t.Fatalf("expected triggers, got:\n%s", catalog)
	}
	if !strings.Contains(catalog, `Example: "I want a refund for order 42"`) {
		t.Fatalf("expected example, got:\n%s", catalog)
	}
	if !strings.Contains(catalog, `activate_skill({"name":"billing"})`) {
		t.Fatalf("expected activate hint, got:\n%s", catalog)
	}
	if !strings.Contains(catalog, "2. **support**") {
		t.Fatalf("expected support entry, got:\n%s", catalog)
	}
}

func TestSkillCatalogOmitsEmptyTriggers(t *testing.T) {
	catalog := SkillCatalog(SkillCatalogData{
		Title:             SkillCatalogTitle,
		ActivateSkillTool: "activate_skill",
		Skills: []SkillRow{
			{Name: "support", Description: "Answer product questions."},
		},
	})

	if strings.Contains(catalog, "Use when:") {
		t.Fatalf("expected no triggers line, got:\n%s", catalog)
	}
}

func TestActivateSkillToolDescription(t *testing.T) {
	desc := ActivateSkillToolDescription(ActivateSkillToolDescriptionData{CatalogTitle: SkillCatalogTitle})
	if !strings.Contains(desc, SkillCatalogTitle) {
		t.Fatalf("expected catalog title in description, got %q", desc)
	}
	if !strings.Contains(desc, "routing list") {
		t.Fatalf("expected routing list reference, got %q", desc)
	}
}

func TestKnowledgeEntry(t *testing.T) {
	got := KnowledgeEntry(KnowledgeEntryData{
		Name:    "policy",
		Content: "Be helpful.",
		Source:  "/facts/policy.md",
	})
	want := "[policy] Be helpful. (source: /facts/policy.md)"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestToolLocked(t *testing.T) {
	got := ToolLocked(ToolLockedData{
		ToolName:          "issue_refund",
		SkillName:         "billing",
		SkillDescription:  "Handle refunds and billing inquiries.",
		ActivateSkillTool: "activate_skill",
	})
	want := `tool "issue_refund" requires the "billing" skill (Handle refunds and billing inquiries.). Call activate_skill({"name":"billing"}) before using this tool.`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestToolLockedWithoutDescription(t *testing.T) {
	got := ToolLocked(ToolLockedData{
		ToolName:          "issue_refund",
		SkillName:         "billing",
		ActivateSkillTool: "activate_skill",
	})
	want := `tool "issue_refund" requires the "billing" skill. Call activate_skill({"name":"billing"}) before using this tool.`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
