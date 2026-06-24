package chromem

import (
	"context"
	"fmt"
	"maps"
	"runtime"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"
	"github.com/clubpay/ronykit/x/rkit"
	chromem "github.com/philippgille/chromem-go"
)

// Config configures an embedded chromem knowledge store.
type Config struct {
	Collection string
	Embedder   intent.Embedder
	PersistDir string
}

// Store provides RAG retrieval and indexing over chromem-go.
type Store struct {
	collection *chromem.Collection
	embedder   intent.Embedder
}

// New creates or opens a chromem collection for knowledge indexing and retrieval.
func New(cfg Config) (*Store, error) {
	if cfg.Embedder == nil {
		return nil, fmt.Errorf("chromem knowledge: embedder is required")
	}

	name := cfg.Collection
	if name == "" {
		name = "intent-knowledge"
	}

	var (
		db  *chromem.DB
		err error
	)
	if cfg.PersistDir != "" {
		db, err = chromem.NewPersistentDB(cfg.PersistDir, false)
		if err != nil {
			return nil, fmt.Errorf("chromem knowledge: open persistent db: %w", err)
		}
	} else {
		db = chromem.NewDB()
	}

	coll, err := db.GetOrCreateCollection(name, nil, toChromemEmbedder(cfg.Embedder))
	if err != nil {
		return nil, fmt.Errorf("chromem knowledge: create collection: %w", err)
	}

	return &Store{collection: coll, embedder: cfg.Embedder}, nil
}

func (s *Store) List(_ context.Context, _ intent.Filter) ([]intent.Entry, error) {
	return nil, errs.ErrUnsupportedOperation
}

func (s *Store) Get(ctx context.Context, id string) (intent.Entry, error) {
	if s == nil || s.collection == nil {
		return intent.Entry{}, errs.ErrKnowledgeNotFound
	}

	doc, err := s.collection.GetByID(ctx, id)
	if err != nil {
		return intent.Entry{}, errs.ErrKnowledgeNotFound
	}

	return documentToEntry(doc), nil
}

func (s *Store) Retrieve(ctx context.Context, q intent.RetrieveQuery) ([]intent.Entry, error) {
	if s == nil || s.collection == nil {
		return nil, nil
	}

	if q.Text == "" {
		return nil, errs.Wrap(errs.ErrUnsupportedOperation, "retrieve query text is empty")
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	results, err := s.collection.Query(ctx, q.Text, limit, q.Filter, nil)
	if err != nil {
		return nil, fmt.Errorf("chromem retrieve: %w", err)
	}

	entries := make([]intent.Entry, 0, len(results))
	for _, res := range results {
		entry := resultToEntry(res)
		if q.MinScore > 0 && entry.Score < q.MinScore {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *Store) Index(ctx context.Context, docs []intent.Document) error {
	if s == nil || s.collection == nil {
		return errs.Wrap(errs.ErrUnsupportedOperation, "chromem store is nil")
	}

	if len(docs) == 0 {
		return nil
	}

	chromemDocs := make([]chromem.Document, 0, len(docs))
	for _, doc := range docs {
		id := doc.ID
		if id == "" {
			id = rkit.RandomID(12)
		}

		meta := map[string]string{
			"title":     doc.Title,
			"source":    doc.Source,
			"mime_type": doc.MIMEType,
		}
		maps.Copy(meta, doc.Meta)

		chromemDocs = append(chromemDocs, chromem.Document{
			ID:       id,
			Content:  doc.Content,
			Metadata: meta,
		})
	}

	err := s.collection.AddDocuments(ctx, chromemDocs, runtime.NumCPU())
	if err != nil {
		return fmt.Errorf("chromem index: %w", err)
	}

	return nil
}

func (s *Store) DeleteSource(ctx context.Context, source string) error {
	if s == nil || s.collection == nil {
		return errs.Wrap(errs.ErrUnsupportedOperation, "chromem store is nil")
	}

	if source == "" {
		return errs.Wrap(errs.ErrUnsupportedOperation, "source is empty")
	}

	err := s.collection.Delete(ctx, map[string]string{"source": source}, nil)
	if err != nil {
		return fmt.Errorf("chromem delete source: %w", err)
	}

	return nil
}

var (
	_ intent.Knowledge = (*Store)(nil)
	_ intent.Indexer   = (*Store)(nil)
	_ intent.Retriever = (*Store)(nil)
)

func toChromemEmbedder(e intent.Embedder) chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		vecs, err := e.Embed(ctx, []string{text})
		if err != nil {
			return nil, err
		}

		if len(vecs) == 0 {
			return nil, fmt.Errorf("embedder returned no vectors")
		}

		return vecs[0], nil
	}
}

func documentToEntry(doc chromem.Document) intent.Entry {
	return intent.Entry{
		ID:      doc.ID,
		Kind:    intent.KindChunk,
		Origin:  intent.OriginDynamic,
		Name:    doc.Metadata["title"],
		Content: doc.Content,
		Source:  doc.Metadata["source"],
		Meta:    doc.Metadata,
	}
}

func resultToEntry(res chromem.Result) intent.Entry {
	return intent.Entry{
		ID:      res.ID,
		Kind:    intent.KindChunk,
		Origin:  intent.OriginDynamic,
		Name:    res.Metadata["title"],
		Content: res.Content,
		Source:  res.Metadata["source"],
		Score:   float64(res.Similarity),
		Meta:    res.Metadata,
	}
}
