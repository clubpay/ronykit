package inmem

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"
	"github.com/clubpay/ronykit/x/rkit"
)

// Memory is an in-memory session-scoped memory backend.
type Memory struct {
	mu       sync.RWMutex
	sessions map[string]*sessionStore
}

type sessionStore struct {
	mu      sync.RWMutex
	records map[string]intent.MemoryRecord
}

// New returns an empty in-memory memory store.
func New() *Memory {
	return &Memory{sessions: make(map[string]*sessionStore)}
}

func (m *Memory) ForSession(sessionID string) intent.SessionMemory {
	m.mu.Lock()
	defer m.mu.Unlock()

	store, ok := m.sessions[sessionID]
	if !ok {
		store = &sessionStore{records: make(map[string]intent.MemoryRecord)}
		m.sessions[sessionID] = store
	}

	return &sessionMemory{sessionID: sessionID, store: store}
}

type sessionMemory struct {
	sessionID string
	store     *sessionStore
}

func (s *sessionMemory) SessionID() string { return s.sessionID }

func (s *sessionMemory) Save(_ context.Context, rec intent.MemoryRecord) error {
	if s == nil || s.store == nil {
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

	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	s.store.records[rec.ID] = rec

	return nil
}

func (s *sessionMemory) Search(_ context.Context, q intent.MemoryQuery) ([]intent.MemoryResult, error) {
	if s == nil || s.store == nil {
		return nil, nil
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	var results []intent.MemoryResult

	for _, rec := range s.store.records {
		if !recordMatches(rec, q) {
			continue
		}

		score := scoreRecord(rec, q)
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

func (s *sessionMemory) Delete(_ context.Context, id string) error {
	if id == "" {
		return errs.Wrap(errs.ErrRecordNotFound, "record id is empty")
	}

	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	if _, ok := s.store.records[id]; !ok {
		return errs.Wrap(errs.ErrRecordNotFound, "record not found")
	}

	delete(s.store.records, id)

	return nil
}

func (s *sessionMemory) Clear(_ context.Context) error {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	s.store.records = make(map[string]intent.MemoryRecord)

	return nil
}

var _ intent.Memory = (*Memory)(nil)

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

func scoreRecord(rec intent.MemoryRecord, q intent.MemoryQuery) float64 {
	if q.Text == "" {
		return 1
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

func sortResults(results []intent.MemoryResult) {
	for i := range results {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}
