package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"
	"github.com/clubpay/ronykit/x/rkit"
)

// Dialect selects SQL differences between supported databases.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

// Memory is a SQL-backed session memory store.
type Memory struct {
	db       *sql.DB
	dialect  Dialect
	embedder intent.Embedder
}

// New opens a Memory store over db. migrate must be true on first use to create tables.
func New(ctx context.Context, db *sql.DB, dialect Dialect, embedder intent.Embedder, migrate bool) (*Memory, error) {
	if db == nil {
		return nil, fmt.Errorf("sqlstore: database is nil")
	}

	m := &Memory{db: db, dialect: dialect, embedder: embedder}
	if migrate {
		err := m.migrate(ctx)
		if err != nil {
			return nil, err
		}
	}

	return m, nil
}

func (m *Memory) ForSession(sessionID string) intent.SessionMemory {
	return &sessionMemory{parent: m, sessionID: sessionID}
}

type sessionMemory struct {
	parent    *Memory
	sessionID string
}

func (s *sessionMemory) SessionID() string { return s.sessionID }

func (s *sessionMemory) Save(ctx context.Context, rec intent.MemoryRecord) error {
	if s == nil || s.parent == nil {
		return errs.Wrap(errs.ErrUnsupportedOperation, "session memory is nil")
	}

	if rec.ID == "" {
		rec.ID = rkit.RandomID(12)
	}

	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}

	rec.UpdatedAt = now

	if len(rec.Embedding) == 0 && s.parent.embedder != nil && len(rec.Content) > 0 {
		vecs, err := s.parent.embedder.Embed(ctx, []string{string(rec.Content)})
		if err != nil {
			return fmt.Errorf("embed memory record: %w", err)
		}

		if len(vecs) > 0 {
			rec.Embedding = vecs[0]
		}
	}

	metaJSON, err := json.Marshal(rec.Metadata)
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}

	embJSON, err := encodeEmbedding(rec.Embedding)
	if err != nil {
		return err
	}

	_, err = s.parent.db.ExecContext(ctx, s.parent.upsertSQL(),
		s.sessionID,
		rec.ID,
		rec.Key,
		rec.Content,
		string(metaJSON),
		embJSON,
		rec.CreatedAt.UTC().Format(time.RFC3339Nano),
		rec.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("save memory record: %w", err)
	}

	return nil
}

func (s *sessionMemory) Search(ctx context.Context, q intent.MemoryQuery) ([]intent.MemoryResult, error) {
	if s == nil || s.parent == nil {
		return nil, nil
	}

	records, err := s.parent.loadSessionRecords(ctx, s.sessionID)
	if err != nil {
		return nil, err
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	var queryEmbedding []float32

	if q.Text != "" && s.parent.embedder != nil {
		vecs, err := s.parent.embedder.Embed(ctx, []string{q.Text})
		if err != nil {
			return nil, fmt.Errorf("embed query: %w", err)
		}

		if len(vecs) > 0 {
			queryEmbedding = vecs[0]
		}
	}

	var results []intent.MemoryResult

	for _, rec := range records {
		if !recordMatches(rec, q) {
			continue
		}

		score := scoreRecord(rec, q, queryEmbedding)
		if q.MinScore > 0 && score < q.MinScore {
			continue
		}

		results = append(results, intent.MemoryResult{Record: rec, Score: score})
	}

	sortResults(results)

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (s *sessionMemory) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errs.Wrap(errs.ErrRecordNotFound, "record id is empty")
	}

	res, err := s.parent.db.ExecContext(ctx,
		`DELETE FROM intent_memory_records WHERE session_id = ? AND id = ?`,
		s.sessionID, id,
	)
	if err != nil {
		return fmt.Errorf("delete memory record: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return errs.Wrap(errs.ErrRecordNotFound, "record not found")
	}

	return nil
}

func (s *sessionMemory) Clear(ctx context.Context) error {
	_, err := s.parent.db.ExecContext(ctx,
		`DELETE FROM intent_memory_records WHERE session_id = ?`,
		s.sessionID,
	)
	if err != nil {
		return fmt.Errorf("clear session memory: %w", err)
	}

	return nil
}

func (m *Memory) loadSessionRecords(ctx context.Context, sessionID string) ([]intent.MemoryRecord, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT id, key_name, content, metadata, embedding, created_at, updated_at
		FROM intent_memory_records
		WHERE session_id = ?
		ORDER BY updated_at DESC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session records: %w", err)
	}
	defer rows.Close()

	var records []intent.MemoryRecord

	for rows.Next() {
		var (
			rec       intent.MemoryRecord
			metaJSON  string
			embJSON   sql.NullString
			createdAt string
			updatedAt string
		)

		err := rows.Scan(&rec.ID, &rec.Key, &rec.Content, &metaJSON, &embJSON, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan memory record: %w", err)
		}

		rec.Metadata = map[string]string{}
		if metaJSON != "" {
			_ = json.Unmarshal([]byte(metaJSON), &rec.Metadata)
		}

		if embJSON.Valid {
			rec.Embedding, err = decodeEmbedding(embJSON.String)
			if err != nil {
				return nil, err
			}
		}

		rec.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}

		rec.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse updated_at: %w", err)
		}

		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (m *Memory) migrate(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, m.createTableSQL())
	if err != nil {
		return fmt.Errorf("migrate memory schema: %w", err)
	}

	_, err = m.db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_intent_memory_session_key
		ON intent_memory_records (session_id, key_name)`)
	if err != nil {
		return fmt.Errorf("create memory index: %w", err)
	}

	return nil
}

func (m *Memory) createTableSQL() string {
	switch m.dialect {
	case DialectPostgres:
		return `
CREATE TABLE IF NOT EXISTS intent_memory_records (
	session_id TEXT NOT NULL,
	id TEXT NOT NULL,
	key_name TEXT NOT NULL DEFAULT '',
	content BYTEA NOT NULL,
	metadata JSONB NOT NULL DEFAULT '{}',
	embedding JSONB,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (session_id, id)
)`
	default:
		return `
CREATE TABLE IF NOT EXISTS intent_memory_records (
	session_id TEXT NOT NULL,
	id TEXT NOT NULL,
	key_name TEXT NOT NULL DEFAULT '',
	content BLOB NOT NULL,
	metadata TEXT NOT NULL DEFAULT '{}',
	embedding TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (session_id, id)
)`
	}
}

func (m *Memory) upsertSQL() string {
	switch m.dialect {
	case DialectPostgres:
		return `
INSERT INTO intent_memory_records (session_id, id, key_name, content, metadata, embedding, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::timestamptz, $8::timestamptz)
ON CONFLICT (session_id, id) DO UPDATE SET
	key_name = EXCLUDED.key_name,
	content = EXCLUDED.content,
	metadata = EXCLUDED.metadata,
	embedding = EXCLUDED.embedding,
	updated_at = EXCLUDED.updated_at`
	default:
		return `
INSERT INTO intent_memory_records (session_id, id, key_name, content, metadata, embedding, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (session_id, id) DO UPDATE SET
	key_name = excluded.key_name,
	content = excluded.content,
	metadata = excluded.metadata,
	embedding = excluded.embedding,
	updated_at = excluded.updated_at`
	}
}

var _ intent.Memory = (*Memory)(nil)

func encodeEmbedding(vec []float32) (sql.NullString, error) {
	if len(vec) == 0 {
		return sql.NullString{}, nil
	}

	bb, err := json.Marshal(vec)
	if err != nil {
		return sql.NullString{}, fmt.Errorf("encode embedding: %w", err)
	}

	return sql.NullString{String: string(bb), Valid: true}, nil
}

func decodeEmbedding(raw string) ([]float32, error) {
	var vec []float32

	err := json.Unmarshal([]byte(raw), &vec)
	if err != nil {
		return nil, fmt.Errorf("decode embedding: %w", err)
	}

	return vec, nil
}

func recordMatches(rec intent.MemoryRecord, q intent.MemoryQuery) bool {
	for k, v := range q.Filter {
		switch k {
		case "key":
			if rec.Key != v {
				return false
			}
		default:
			if rec.Metadata[k] != v {
				return false
			}
		}
	}

	if q.Text == "" {
		return true
	}

	text := strings.ToLower(q.Text)
	content := strings.ToLower(string(rec.Content))
	key := strings.ToLower(rec.Key)

	return strings.Contains(content, text) || strings.Contains(key, text)
}

func scoreRecord(rec intent.MemoryRecord, q intent.MemoryQuery, queryEmbedding []float32) float64 {
	if q.Text == "" {
		return 1
	}

	if len(queryEmbedding) > 0 && len(rec.Embedding) > 0 {
		return float64(cosineSimilarity(queryEmbedding, rec.Embedding))
	}

	text := strings.ToLower(q.Text)

	content := strings.ToLower(string(rec.Content))
	switch {
	case content == text:
		return 1
	case strings.Contains(content, text):
		return 0.8
	case strings.Contains(strings.ToLower(rec.Key), text):
		return 0.5
	default:
		return 0
	}
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

func sortResults(results []intent.MemoryResult) {
	for i := range results {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}
