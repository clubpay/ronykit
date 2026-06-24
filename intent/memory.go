package intent

import (
	"context"
	"time"
)

// Memory vs knowledge:
//
//   - Memory: session-scoped state — conversation history, tool outputs, and
//     notes for one running session. Use Memory.ForSession.
//
//   - Knowledge: agent knowledge — static catalog plus dynamic RAG over shared
//     corpora. Not tied to a single session.

// MemoryRecord is a stored memory item within a session.
type MemoryRecord struct {
	ID        string
	Key       string
	Content   []byte
	Metadata  map[string]string
	Embedding []float32
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MemoryQuery describes a memory lookup within a session.
type MemoryQuery struct {
	Text     string
	Filter   map[string]string
	Limit    int
	MinScore float64
}

// MemoryResult is a memory search hit.
type MemoryResult struct {
	Record MemoryRecord
	Score  float64
}

// SessionMemory stores and retrieves context for a single agent session.
// Conversation turns, tool results, and task notes for one session belong here.
type SessionMemory interface {
	SessionID() string
	Save(ctx context.Context, rec MemoryRecord) error
	Search(ctx context.Context, q MemoryQuery) ([]MemoryResult, error)
	Delete(ctx context.Context, id string) error
	Clear(ctx context.Context) error
}

// Memory provides session-scoped storage backends.
// Implementations may use in-memory storage, Postgres, embedded vector DBs
// (e.g. chromem-go), or external vector stores (e.g. Milvus).
type Memory interface {
	ForSession(sessionID string) SessionMemory
}
