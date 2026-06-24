package intent

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sync"

	"github.com/clubpay/ronykit/intent/errs"
	"github.com/clubpay/ronykit/x/rkit"
)

const historyRecordKey = "intent:history"

// Session is one conversation or task run.
type Session struct {
	ID        string
	Memory    SessionMemory
	Metadata  map[string]string
	Selection Selection
}

// SessionManager creates and tracks agent sessions.
type SessionManager struct {
	mem    Memory
	active map[string]*Session
	closed map[string]struct{}
	mu     sync.RWMutex
	idLen  int
}

// NewSessionManager returns a session manager backed by mem.
func NewSessionManager(mem Memory) *SessionManager {
	return &SessionManager{
		mem:    mem,
		active: make(map[string]*Session),
		closed: make(map[string]struct{}),
		idLen:  16,
	}
}

// SessionOption configures session creation.
type SessionOption func(*sessionCreateConfig)

type sessionCreateConfig struct {
	id        string
	metadata  map[string]string
	selection Selection
}

// SessionWithID sets an explicit session ID.
func SessionWithID(id string) SessionOption {
	return func(cfg *sessionCreateConfig) {
		cfg.id = id
	}
}

// SessionWithMetadata attaches metadata to the session.
func SessionWithMetadata(metadata map[string]string) SessionOption {
	return func(cfg *sessionCreateConfig) {
		cfg.metadata = metadata
	}
}

// SessionWithSelection sets the LLM selection strategy for the session.
func SessionWithSelection(sel Selection) SessionOption {
	return func(cfg *sessionCreateConfig) {
		cfg.selection = sel
	}
}

// Create opens a new session.
func (m *SessionManager) Create(_ context.Context, opts ...SessionOption) (*Session, error) {
	if m == nil || m.mem == nil {
		return nil, errs.Wrap(errs.ErrUnsupportedOperation, "session manager memory is nil")
	}

	cfg := sessionCreateConfig{selection: Selection{Strategy: StrategyFirst}}
	for _, opt := range opts {
		opt(&cfg)
	}

	id := cfg.id
	if id == "" {
		id = rkit.RandomID(m.idLen)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.active[id]; ok {
		return nil, errs.Wrap(errs.ErrUnsupportedOperation, fmt.Sprintf("session %q already exists", id))
	}

	if _, ok := m.closed[id]; ok {
		return nil, errs.Wrap(errs.ErrUnsupportedOperation, fmt.Sprintf("session %q is closed", id))
	}

	meta := make(map[string]string, len(cfg.metadata))
	maps.Copy(meta, cfg.metadata)

	s := &Session{
		ID:        id,
		Memory:    m.mem.ForSession(id),
		Metadata:  meta,
		Selection: cfg.selection,
	}
	m.active[id] = s

	return s, nil
}

// Get returns an active session by ID.
func (m *SessionManager) Get(_ context.Context, id string) (*Session, error) {
	if id == "" {
		return nil, errs.Wrap(errs.ErrSessionNotFound, "empty session id")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.active[id]
	if !ok {
		return nil, errs.SessionNotFound(id)
	}

	return s, nil
}

// Close removes a session from the active set and clears its memory.
func (m *SessionManager) Close(ctx context.Context, id string) error {
	m.mu.Lock()

	s, ok := m.active[id]
	if ok {
		delete(m.active, id)
		m.closed[id] = struct{}{}
	}
	m.mu.Unlock()

	if !ok {
		return errs.SessionNotFound(id)
	}

	if s.Memory != nil {
		err := s.Memory.Clear(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// LoadHistory returns conversation messages stored on the session.
func LoadHistory(ctx context.Context, s *Session) ([]Message, error) {
	if s == nil || s.Memory == nil {
		return nil, nil
	}

	results, err := s.Memory.Search(ctx, MemoryQuery{
		Filter: map[string]string{"key": historyRecordKey},
		Limit:  1,
	})
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	var msgs []Message

	err = json.Unmarshal(results[0].Record.Content, &msgs)
	if err != nil {
		return nil, fmt.Errorf("decode session history: %w", err)
	}

	return msgs, nil
}

// SaveHistory replaces the stored conversation history for the session.
func SaveHistory(ctx context.Context, s *Session, msgs []Message) error {
	if s == nil || s.Memory == nil {
		return nil
	}

	bb, err := json.Marshal(msgs)
	if err != nil {
		return fmt.Errorf("encode session history: %w", err)
	}

	rec := MemoryRecord{Key: historyRecordKey, Content: bb}

	existing, err := s.Memory.Search(ctx, MemoryQuery{
		Filter: map[string]string{"key": historyRecordKey},
		Limit:  1,
	})
	if err != nil {
		return err
	}

	if len(existing) > 0 {
		rec.ID = existing[0].Record.ID
	}

	return s.Memory.Save(ctx, rec)
}

// AppendHistory appends messages to stored session history.
func AppendHistory(ctx context.Context, s *Session, msgs ...Message) error {
	if len(msgs) == 0 {
		return nil
	}

	history, err := LoadHistory(ctx, s)
	if err != nil {
		return err
	}

	history = append(history, msgs...)

	return SaveHistory(ctx, s, history)
}
