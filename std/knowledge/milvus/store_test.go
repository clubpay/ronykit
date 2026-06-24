package milvus_test

import (
	"context"
	"os"
	"testing"

	"github.com/clubpay/ronykit/intent"
	milvusstore "github.com/clubpay/ronykit/std/knowledge/milvus"
)

type stubEmbedder struct {
	vec []float32
}

func (s stubEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, text := range texts {
		switch text {
		case "Why is the sky blue?":
			out[i] = []float32{1, 0, 0}
		case "The sky is blue because of Rayleigh scattering.":
			out[i] = []float32{0.95, 0.05, 0}
		default:
			out[i] = append([]float32(nil), s.vec...)
		}
	}

	return out, nil
}

func (s stubEmbedder) Dimensions() int { return len(s.vec) }

func TestMilvusIndexAndRetrieve(t *testing.T) {
	addr := os.Getenv("INTENT_KNOWLEDGE_MILVUS_ADDR")
	if addr == "" {
		t.Skip("INTENT_KNOWLEDGE_MILVUS_ADDR not set")
	}

	store, err := milvusstore.Open(context.Background(), milvusstore.Config{
		Address:    addr,
		APIKey:     os.Getenv("INTENT_KNOWLEDGE_MILVUS_API_KEY"),
		Collection: "intent_knowledge_test",
		Embedder:   stubEmbedder{vec: []float32{0, 1, 0}},
		Dimensions: 3,
		AutoCreate: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	err = store.DeleteSource(context.Background(), "science")
	if err != nil {
		t.Fatal(err)
	}

	err = store.Index(context.Background(), []intent.Document{
		{ID: "1", Source: "science", Title: "Sky", Content: "The sky is blue because of Rayleigh scattering."},
	})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := store.Retrieve(context.Background(), intent.RetrieveQuery{
		Text:  "Why is the sky blue?",
		Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != "1" {
		t.Fatalf("unexpected retrieve result: %#v", entries)
	}
}
