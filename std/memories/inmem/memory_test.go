package inmem_test

import (
	"context"
	"errors"
	"testing"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"
	"github.com/clubpay/ronykit/std/memories/inmem"
)

func TestSessionIsolation(t *testing.T) {
	store := inmem.New()

	s1 := store.ForSession("a")
	s2 := store.ForSession("b")

	err := s1.Save(context.Background(), intent.MemoryRecord{Key: "note", Content: []byte("alpha")})
	if err != nil {
		t.Fatal(err)
	}

	results, err := s2.Search(context.Background(), intent.MemoryQuery{Text: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected session isolation, got %#v", results)
	}
}

func TestSearchFilterAndScore(t *testing.T) {
	s := inmem.New().ForSession("s")

	err := s.Save(context.Background(), intent.MemoryRecord{Key: "greeting", Content: []byte("hello world")})
	if err != nil {
		t.Fatal(err)
	}

	results, err := s.Search(context.Background(), intent.MemoryQuery{Text: "hello", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Score <= 0 {
		t.Fatalf("expected positive score, got %f", results[0].Score)
	}
}

func TestDeleteMissingRecord(t *testing.T) {
	s := inmem.New().ForSession("s")
	err := s.Delete(context.Background(), "missing")
	if !errors.Is(err, errs.ErrRecordNotFound) {
		t.Fatalf("expected record not found, got %v", err)
	}
}

func TestClearRemovesRecords(t *testing.T) {
	s := inmem.New().ForSession("s")
	err := s.Save(context.Background(), intent.MemoryRecord{Key: "x", Content: []byte("y")})
	if err != nil {
		t.Fatal(err)
	}

	err = s.Clear(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	results, err := s.Search(context.Background(), intent.MemoryQuery{Text: "y"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected empty session after clear")
	}
}
