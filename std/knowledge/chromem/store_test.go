package chromem_test

import (
	"context"
	"errors"
	"testing"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"
	chromestore "github.com/clubpay/ronykit/std/knowledge/chromem"
)

type stubEmbedder struct {
	vec []float32
}

func (s stubEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, text := range texts {
		out[i] = s.vectorFor(text)
	}

	return out, nil
}

func (s stubEmbedder) Dimensions() int { return len(s.vec) }

func (s stubEmbedder) vectorFor(text string) []float32 {
	switch text {
	case "Why is the sky blue?":
		return []float32{1, 0, 0}
	case "The sky is blue because of Rayleigh scattering.":
		return []float32{0.95, 0.05, 0}
	case "Leaves are green because chlorophyll absorbs red and blue light.":
		return []float32{0, 1, 0}
	default:
		return append([]float32(nil), s.vec...)
	}
}

func TestIndexAndRetrieve(t *testing.T) {
	store, err := chromestore.New(chromestore.Config{
		Embedder: stubEmbedder{vec: []float32{0, 0, 1}},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = store.Index(context.Background(), []intent.Document{
		{ID: "1", Source: "science", Title: "Sky", Content: "The sky is blue because of Rayleigh scattering."},
		{ID: "2", Source: "science", Title: "Leaves", Content: "Leaves are green because chlorophyll absorbs red and blue light."},
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
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID != "1" {
		t.Fatalf("unexpected top hit: %#v", entries[0])
	}
}

func TestDeleteSource(t *testing.T) {
	store, err := chromestore.New(chromestore.Config{
		Embedder: stubEmbedder{vec: []float32{0, 0, 1}},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = store.Index(context.Background(), []intent.Document{
		{ID: "1", Source: "docs/a", Content: "alpha"},
		{ID: "2", Source: "docs/b", Content: "beta"},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = store.DeleteSource(context.Background(), "docs/a")
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Get(context.Background(), "1")
	if !errors.Is(err, errs.ErrKnowledgeNotFound) {
		t.Fatalf("expected deleted document to be missing, got %v", err)
	}
}

func TestListUnsupported(t *testing.T) {
	store, err := chromestore.New(chromestore.Config{
		Embedder: stubEmbedder{vec: []float32{1}},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.List(context.Background(), intent.Filter{})
	if !errors.Is(err, errs.ErrUnsupportedOperation) {
		t.Fatalf("expected unsupported, got %v", err)
	}
}
