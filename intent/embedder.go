package intent

import "context"

// Embedder produces vector embeddings for text.
// Adapters target github.com/tmc/langchaingo/embeddings.Embedder.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimensions() int
}
