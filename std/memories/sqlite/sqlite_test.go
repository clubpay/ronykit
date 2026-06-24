package sqlite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"
	"github.com/clubpay/ronykit/std/memories/sqlite"
)

func TestSQLiteMemorySessionIsolation(t *testing.T) {
	store, err := sqlite.Open(context.Background(), sqlite.Config{
		DSN:     "file:" + t.Name() + "?mode=memory&cache=shared",
		Migrate: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	s1 := store.ForSession("a")
	s2 := store.ForSession("b")

	err = s1.Save(context.Background(), intent.MemoryRecord{Key: "note", Content: []byte("alpha")})
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

func TestSQLiteMemoryDeleteMissingRecord(t *testing.T) {
	store, err := sqlite.Open(context.Background(), sqlite.Config{
		DSN:     "file:" + t.Name() + "?mode=memory&cache=shared",
		Migrate: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	err = store.ForSession("s").Delete(context.Background(), "missing")
	if !errors.Is(err, errs.ErrRecordNotFound) {
		t.Fatalf("expected record not found, got %v", err)
	}
}
