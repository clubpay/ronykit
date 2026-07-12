package langchaingo

import (
	"context"

	"github.com/clubpay/ronykit/intent"

	lcemb "github.com/tmc/langchaingo/embeddings"
)

// Embedder adapts langchaingo embeddings.Embedder to intent.Embedder.
type Embedder struct {
	inner      lcemb.Embedder
	dimensions int
}

// New wraps a langchaingo embedder. dimensions must match the model output size.
func New(inner lcemb.Embedder, dimensions int) *Embedder {
	return &Embedder{inner: inner, dimensions: dimensions}
}

func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if e == nil || e.inner == nil {
		return nil, nil
	}

	if len(texts) == 0 {
		return nil, nil
	}

	vecs, err := e.inner.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, err
	}

	return vecs, nil
}

func (e *Embedder) Dimensions() int {
	if e == nil {
		return 0
	}

	return e.dimensions
}

var _ intent.Embedder = (*Embedder)(nil)
