package intent

import "context"

// Knowledge layers:
//
//   - Static (StaticStore): prompts, skills, and facts configured on the agent.
//     Loaded at setup; stable across sessions.
//
//   - Dynamic (Retriever): RAG retrieval over indexed document corpora at
//     request time. Returns scored chunks/documents (OriginDynamic).
//
//   - Indexer: optional ingestion path for dynamic backends.
//
// Session-scoped conversation state lives in Memory, not here.

// Kind identifies the type of knowledge entry.
type Kind string

const (
	KindPrompt   Kind = "prompt"
	KindSkill    Kind = "skill"
	KindFact     Kind = "fact"
	KindDocument Kind = "document"
	KindChunk    Kind = "chunk"
)

// Origin distinguishes configured knowledge from RAG-retrieved knowledge.
type Origin string

const (
	OriginStatic  Origin = "static"
	OriginDynamic Origin = "dynamic"
)

// Entry is a knowledge item from the static catalog or a RAG retrieval hit.
type Entry struct {
	ID      string
	Kind    Kind
	Origin  Origin
	Name    string
	Content string
	Source  string // document URI, file path, collection name, etc.
	Score   float64
	Meta    map[string]string
}

// Filter restricts static List results.
type Filter struct {
	Kinds []Kind
	Names []string
}

// RetrieveQuery describes a dynamic RAG lookup over indexed corpora.
type RetrieveQuery struct {
	Text     string
	Filter   map[string]string
	Limit    int
	MinScore float64
}

// StaticStore holds configured prompts, skills, and facts.
// These entries are loaded at setup time and do not change during a session.
type StaticStore interface {
	List(ctx context.Context, filter Filter) ([]Entry, error)
	Get(ctx context.Context, id string) (Entry, error)
}

// Retriever performs dynamic knowledge lookup (RAG) at the request time.
// Implementations may use embedded vector DBs (e.g. chromem-go) or external
// stores (e.g. Milvus) over document corpora.
type Retriever interface {
	Retrieve(ctx context.Context, q RetrieveQuery) ([]Entry, error)
}

// Knowledge combines the static catalog and dynamic RAG retrieval.
// Implementations may embed only one side; callers should handle unsupported
// operations from partial implementations as needed.
type Knowledge interface {
	StaticStore
	Retriever
}

// Indexer ingests documents into a dynamic knowledge backend.
// Separated from Retriever so ingestion pipelines can run independently of queries.
type Indexer interface {
	Index(ctx context.Context, docs []Document) error
	DeleteSource(ctx context.Context, source string) error
}

// Document is raw content to index for RAG retrieval.
type Document struct {
	ID       string
	Source   string
	Title    string
	Content  string
	MIMEType string
	Meta     map[string]string
}
