package langchaingo_test

import (
	"context"
	"testing"

	"github.com/clubpay/ronykit/std/embedders/langchaingo"
	lcemb "github.com/tmc/langchaingo/embeddings"
)

type stubEmbedder struct {
	vecs [][]float32
}

func (s stubEmbedder) EmbedDocuments(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		if i < len(s.vecs) {
			out[i] = s.vecs[i]
		}
	}

	return out, nil
}

func (s stubEmbedder) EmbedQuery(_ context.Context, _ string) ([]float32, error) {
	return nil, nil
}

var _ lcemb.Embedder = stubEmbedder{}

func TestEmbedder(t *testing.T) {
	inner := stubEmbedder{vecs: [][]float32{{0.1, 0.2, 0.3}}}
	emb := langchaingo.New(inner, 3)

	vecs, err := emb.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vecs) != 1 || len(vecs[0]) != 3 {
		t.Fatalf("unexpected vectors: %#v", vecs)
	}
	if emb.Dimensions() != 3 {
		t.Fatalf("expected dimensions 3, got %d", emb.Dimensions())
	}
}
