package intent

import (
	"strings"
	"testing"
)

func TestFormatSkillCatalogUsesPrompt(t *testing.T) {
	catalog := FormatSkillCatalog([]SkillCard{
		{
			Name:        "billing",
			Description: "Handle refunds and billing inquiries.",
			Triggers:    []string{"refund"},
			Examples:    []string{"refund order 42"},
		},
	})

	if !strings.Contains(catalog, "## Skill Catalog") {
		t.Fatalf("expected Skill Catalog heading, got:\n%s", catalog)
	}
	if !strings.Contains(catalog, "**billing**") {
		t.Fatalf("expected billing entry, got:\n%s", catalog)
	}
	if !strings.Contains(catalog, "Use when: refund") {
		t.Fatalf("expected triggers, got:\n%s", catalog)
	}
}

func TestActivateSkillDefinitionReferencesCatalog(t *testing.T) {
	def := activateSkillDefinition([]SkillCard{
		{Name: "billing", Description: "Handle refunds."},
	})

	if !strings.Contains(def.Description, SkillCatalogTitle) {
		t.Fatalf("tool description should reference catalog, got %q", def.Description)
	}
	if !strings.Contains(def.Description, "routing list") {
		t.Fatalf("tool description should reference routing list, got %q", def.Description)
	}

	props, ok := def.Parameters.(map[string]any)["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties map")
	}

	nameDesc, ok := props["name"].(map[string]any)["description"].(string)
	if !ok {
		t.Fatal("expected name parameter description")
	}
	if !strings.Contains(nameDesc, SkillCatalogTitle) {
		t.Fatalf("name parameter should reference catalog, got %q", nameDesc)
	}
}
