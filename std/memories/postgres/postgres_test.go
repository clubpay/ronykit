package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/std/memories/postgres"
)

func TestPostgresMemory(t *testing.T) {
	dsn := os.Getenv("INTENT_MEMORY_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("INTENT_MEMORY_POSTGRES_DSN not set")
	}

	store, err := postgres.Open(context.Background(), postgres.Config{
		DSN:     dsn,
		Migrate: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	s := store.ForSession("test-session")
	err = s.Save(context.Background(), intent.MemoryRecord{Key: "greeting", Content: []byte("hello postgres")})
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
}
