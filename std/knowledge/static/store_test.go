package static_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"
	"github.com/clubpay/ronykit/std/knowledge/static"
)

func TestLoadDirAndGet(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "prompts", "system.md"), "be helpful")
	write(t, filepath.Join(root, "facts", "product.md"), "widget")

	store, err := static.LoadDir(root)
	if err != nil {
		t.Fatal(err)
	}

	entry, err := store.Get(context.Background(), "prompts/system")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Kind != intent.KindPrompt || entry.Content != "be helpful" {
		t.Fatalf("unexpected entry: %#v", entry)
	}

	entries, err := store.List(context.Background(), intent.Filter{Kinds: []intent.Kind{intent.KindFact}})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "product" {
		t.Fatalf("unexpected facts: %#v", entries)
	}
}

func TestGetMissing(t *testing.T) {
	store, err := static.LoadDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Get(context.Background(), "missing")
	if !errors.Is(err, errs.ErrKnowledgeNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestRetrieveUnsupported(t *testing.T) {
	store, err := static.LoadDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Retrieve(context.Background(), intent.RetrieveQuery{Text: "x"})
	if !errors.Is(err, errs.ErrUnsupportedOperation) {
		t.Fatalf("expected unsupported, got %v", err)
	}
}

func TestLoadDirSkillsFrontMatter(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "prompts", "system.md"), "be helpful")
	write(t, filepath.Join(root, "skills", "billing.md"), `---
name: billing
description: Handle refunds and billing inquiries.
tools:
  - issue_refund
  - lookup_order
triggers:
  - refund
  - chargeback
examples:
  - "I want a refund for order 42"
---
Refund policy: verify the order first.`)

	store, err := static.LoadDir(root)
	if err != nil {
		t.Fatal(err)
	}

	cards, err := store.Skills().List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || cards[0].Name != "billing" {
		t.Fatalf("unexpected cards: %+v", cards)
	}
	if cards[0].Description != "Handle refunds and billing inquiries." {
		t.Fatalf("description not parsed: %q", cards[0].Description)
	}

	skill, err := store.Skills().Get(context.Background(), "billing")
	if err != nil {
		t.Fatal(err)
	}
	if skill.Instructions != "Refund policy: verify the order first." {
		t.Fatalf("instructions not parsed: %q", skill.Instructions)
	}
	if len(skill.Tools) != 2 || skill.Tools[0] != "issue_refund" {
		t.Fatalf("tools not parsed: %v", skill.Tools)
	}
	if len(skill.Triggers) != 2 || skill.Triggers[0] != "refund" {
		t.Fatalf("triggers not parsed: %v", skill.Triggers)
	}
	if len(skill.Examples) != 1 || skill.Examples[0] != "I want a refund for order 42" {
		t.Fatalf("examples not parsed: %v", skill.Examples)
	}
	if cards[0].Triggers == nil || cards[0].Triggers[0] != "refund" {
		t.Fatalf("triggers not on card: %+v", cards[0])
	}

	// Skills are not eagerly listed as static entries; prompts still are.
	prompts, err := store.List(context.Background(), intent.Filter{Kinds: []intent.Kind{intent.KindPrompt}})
	if err != nil {
		t.Fatal(err)
	}
	if len(prompts) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(prompts))
	}
}

func TestLoadDirSkillWithoutFrontMatter(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "skills", "greet.md"), "Always greet warmly.")

	store, err := static.LoadDir(root)
	if err != nil {
		t.Fatal(err)
	}

	skill, err := store.Skills().Get(context.Background(), "greet")
	if err != nil {
		t.Fatal(err)
	}
	if skill.Name != "greet" || skill.Instructions != "Always greet warmly." {
		t.Fatalf("unexpected skill: %+v", skill)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(path, []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}
