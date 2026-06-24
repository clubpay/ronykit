package milvus

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"
	"github.com/clubpay/ronykit/x/rkit"
	milvusclient "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	fieldID       = "id"
	fieldSource   = "source"
	fieldTitle    = "title"
	fieldContent  = "content"
	fieldMIMEType = "mime_type"
	fieldMeta     = "meta"
	fieldVector   = "vector"
)

// Config configures a Milvus knowledge store.
type Config struct {
	Client     milvusclient.Client
	Address    string
	Username   string
	Password   string
	APIKey     string
	Collection string
	Embedder   intent.Embedder
	Dimensions int
	AutoCreate bool
}

// Store provides RAG retrieval and indexing over Milvus.
type Store struct {
	client     milvusclient.Client
	collection string
	embedder   intent.Embedder
	dimensions int
	ownsClient bool
}

// Open connects to Milvus and prepares a knowledge store.
func Open(ctx context.Context, cfg Config) (*Store, error) {
	if cfg.Embedder == nil {
		return nil, fmt.Errorf("milvus knowledge: embedder is required")
	}

	if cfg.Dimensions <= 0 {
		return nil, fmt.Errorf("milvus knowledge: dimensions must be positive")
	}

	collection := cfg.Collection
	if collection == "" {
		collection = "intent_knowledge"
	}

	client := cfg.Client
	ownsClient := false

	if client == nil {
		if cfg.Address == "" {
			return nil, fmt.Errorf("milvus knowledge: address or client is required")
		}

		var err error

		client, err = milvusclient.NewClient(ctx, milvusclient.Config{
			Address:  cfg.Address,
			Username: cfg.Username,
			Password: cfg.Password,
			APIKey:   cfg.APIKey,
		})
		if err != nil {
			return nil, fmt.Errorf("milvus knowledge: connect: %w", err)
		}

		ownsClient = true
	}

	store := &Store{
		client:     client,
		collection: collection,
		embedder:   cfg.Embedder,
		dimensions: cfg.Dimensions,
		ownsClient: ownsClient,
	}

	if cfg.AutoCreate {
		err := store.ensureCollection(ctx)
		if err != nil {
			if ownsClient {
				_ = client.Close()
			}

			return nil, err
		}
	}

	return store, nil
}

// Close closes the Milvus client when this store created it.
func (s *Store) Close() error {
	if s == nil || !s.ownsClient || s.client == nil {
		return nil
	}

	return s.client.Close()
}

func (s *Store) List(_ context.Context, _ intent.Filter) ([]intent.Entry, error) {
	return nil, errs.ErrUnsupportedOperation
}

func (s *Store) Get(ctx context.Context, id string) (intent.Entry, error) {
	if s == nil || s.client == nil {
		return intent.Entry{}, errs.ErrKnowledgeNotFound
	}

	if id == "" {
		return intent.Entry{}, errs.ErrKnowledgeNotFound
	}

	result, err := s.client.Query(ctx, s.collection, nil,
		fmt.Sprintf(`%s == %q`, fieldID, id),
		[]string{fieldID, fieldSource, fieldTitle, fieldContent, fieldMIMEType, fieldMeta},
	)
	if err != nil {
		return intent.Entry{}, fmt.Errorf("milvus get: %w", err)
	}

	if result.Len() == 0 {
		return intent.Entry{}, errs.ErrKnowledgeNotFound
	}

	return resultSetToEntry(result, 0), nil
}

func (s *Store) Retrieve(ctx context.Context, q intent.RetrieveQuery) ([]intent.Entry, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}

	if q.Text == "" {
		return nil, errs.Wrap(errs.ErrUnsupportedOperation, "retrieve query text is empty")
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	vecs, err := s.embedder.Embed(ctx, []string{q.Text})
	if err != nil {
		return nil, fmt.Errorf("milvus embed query: %w", err)
	}

	if len(vecs) == 0 {
		return nil, fmt.Errorf("milvus embed query: no vectors returned")
	}

	results, err := s.client.Search(ctx, s.collection, nil,
		filterExpr(q.Filter),
		[]string{fieldID, fieldSource, fieldTitle, fieldContent, fieldMIMEType, fieldMeta},
		[]entity.Vector{entity.FloatVector(vecs[0])},
		fieldVector,
		entity.COSINE,
		limit,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("milvus search: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	result := results[0]

	entries := make([]intent.Entry, 0, result.ResultCount)
	for i := range result.ResultCount {
		entry := searchResultToEntry(result, i)
		if len(result.Scores) > i {
			entry.Score = float64(result.Scores[i])
		}

		if q.MinScore > 0 && entry.Score < q.MinScore {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *Store) Index(ctx context.Context, docs []intent.Document) error {
	if s == nil || s.client == nil {
		return errs.Wrap(errs.ErrUnsupportedOperation, "milvus store is nil")
	}

	if len(docs) == 0 {
		return nil
	}

	ids := make([]string, len(docs))
	sources := make([]string, len(docs))
	titles := make([]string, len(docs))
	contents := make([]string, len(docs))
	mimeTypes := make([]string, len(docs))
	metas := make([]string, len(docs))
	texts := make([]string, len(docs))

	for i, doc := range docs {
		id := doc.ID
		if id == "" {
			id = rkit.RandomID(12)
		}

		ids[i] = id
		sources[i] = doc.Source
		titles[i] = doc.Title
		contents[i] = doc.Content
		mimeTypes[i] = doc.MIMEType
		texts[i] = doc.Content

		meta := map[string]string{}
		maps.Copy(meta, doc.Meta)

		metaJSON, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("encode document meta: %w", err)
		}

		metas[i] = string(metaJSON)
	}

	vectors, err := s.embedder.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("milvus embed documents: %w", err)
	}

	if len(vectors) != len(docs) {
		return fmt.Errorf("milvus embed documents: expected %d vectors, got %d", len(docs), len(vectors))
	}

	_, err = s.client.Insert(ctx, s.collection, "",
		entity.NewColumnVarChar(fieldID, ids),
		entity.NewColumnVarChar(fieldSource, sources),
		entity.NewColumnVarChar(fieldTitle, titles),
		entity.NewColumnVarChar(fieldContent, contents),
		entity.NewColumnVarChar(fieldMIMEType, mimeTypes),
		entity.NewColumnVarChar(fieldMeta, metas),
		entity.NewColumnFloatVector(fieldVector, s.dimensions, vectors),
	)
	if err != nil {
		return fmt.Errorf("milvus insert: %w", err)
	}

	return s.client.Flush(ctx, s.collection, false)
}

func (s *Store) DeleteSource(ctx context.Context, source string) error {
	if s == nil || s.client == nil {
		return errs.Wrap(errs.ErrUnsupportedOperation, "milvus store is nil")
	}

	if source == "" {
		return errs.Wrap(errs.ErrUnsupportedOperation, "source is empty")
	}

	err := s.client.Delete(ctx, s.collection, "", fmt.Sprintf(`%s == %q`, fieldSource, source))
	if err != nil {
		return fmt.Errorf("milvus delete source: %w", err)
	}

	return s.client.Flush(ctx, s.collection, false)
}

var (
	_ intent.Knowledge = (*Store)(nil)
	_ intent.Indexer   = (*Store)(nil)
	_ intent.Retriever = (*Store)(nil)
)

func (s *Store) ensureCollection(ctx context.Context) error {
	exists, err := s.client.HasCollection(ctx, s.collection)
	if err != nil {
		return fmt.Errorf("milvus has collection: %w", err)
	}

	if exists {
		return nil
	}

	schema := entity.NewSchema().
		WithName(s.collection).
		WithField(entity.NewField().
			WithName(fieldID).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(128).
			WithIsPrimaryKey(true),
		).
		WithField(entity.NewField().
			WithName(fieldSource).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(512),
		).
		WithField(entity.NewField().
			WithName(fieldTitle).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(512),
		).
		WithField(entity.NewField().
			WithName(fieldContent).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(65535),
		).
		WithField(entity.NewField().
			WithName(fieldMIMEType).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(128),
		).
		WithField(entity.NewField().
			WithName(fieldMeta).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(8192),
		).
		WithField(entity.NewField().
			WithName(fieldVector).
			WithDataType(entity.FieldTypeFloatVector).
			WithDim(int64(s.dimensions)),
		)

	err = s.client.CreateCollection(ctx, schema, 1)
	if err != nil {
		return fmt.Errorf("milvus create collection: %w", err)
	}

	idx, err := entity.NewIndexAUTOINDEX(entity.COSINE)
	if err != nil {
		return fmt.Errorf("milvus create index params: %w", err)
	}

	err = s.client.CreateIndex(ctx, s.collection, fieldVector, idx, false)
	if err != nil {
		return fmt.Errorf("milvus create index: %w", err)
	}

	return s.client.LoadCollection(ctx, s.collection, false)
}

func filterExpr(filter map[string]string) string {
	if len(filter) == 0 {
		return ""
	}

	allowed := map[string]string{
		fieldID:       fieldID,
		fieldSource:   fieldSource,
		fieldTitle:    fieldTitle,
		fieldMIMEType: fieldMIMEType,
	}

	parts := make([]string, 0, len(filter))
	for key, value := range filter {
		field, ok := allowed[key]
		if !ok {
			continue
		}

		parts = append(parts, fmt.Sprintf(`%s == %q`, field, value))
	}

	return strings.Join(parts, " && ")
}

func resultSetToEntry(rs milvusclient.ResultSet, i int) intent.Entry {
	return columnSetToEntry(func(name string) entity.Column {
		return rs.GetColumn(name)
	}, i)
}

func searchResultToEntry(rs milvusclient.SearchResult, i int) intent.Entry {
	return columnSetToEntry(func(name string) entity.Column {
		return rs.Fields.GetColumn(name)
	}, i)
}

func columnSetToEntry(getCol func(string) entity.Column, i int) intent.Entry {
	entry := intent.Entry{
		Kind:   intent.KindChunk,
		Origin: intent.OriginDynamic,
	}

	if col := getCol(fieldID); col != nil {
		if id, err := col.GetAsString(i); err == nil {
			entry.ID = id
		}
	}

	if col := getCol(fieldTitle); col != nil {
		if title, err := col.GetAsString(i); err == nil {
			entry.Name = title
		}
	}

	if col := getCol(fieldContent); col != nil {
		if content, err := col.GetAsString(i); err == nil {
			entry.Content = content
		}
	}

	if col := getCol(fieldSource); col != nil {
		if source, err := col.GetAsString(i); err == nil {
			entry.Source = source
		}
	}

	meta := map[string]string{}

	if col := getCol(fieldMeta); col != nil {
		if raw, err := col.GetAsString(i); err == nil && raw != "" {
			_ = json.Unmarshal([]byte(raw), &meta)
		}
	}

	if col := getCol(fieldMIMEType); col != nil {
		if mime, err := col.GetAsString(i); err == nil && mime != "" {
			meta["mime_type"] = mime
		}
	}

	if entry.Source != "" {
		meta["source"] = entry.Source
	}

	if entry.Name != "" {
		meta["title"] = entry.Name
	}

	entry.Meta = meta

	return entry
}
