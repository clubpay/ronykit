package intent_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"
)

type fakeMemory struct {
	sessions map[string]*fakeSessionMemory
}

type fakeSessionMemory struct {
	id      string
	records map[string]intent.MemoryRecord
}

func newFakeMemory() *fakeMemory {
	return &fakeMemory{sessions: make(map[string]*fakeSessionMemory)}
}

func (m *fakeMemory) ForSession(id string) intent.SessionMemory {
	s, ok := m.sessions[id]
	if !ok {
		s = &fakeSessionMemory{id: id, records: make(map[string]intent.MemoryRecord)}
		m.sessions[id] = s
	}

	return s
}

func (s *fakeSessionMemory) SessionID() string { return s.id }

func (s *fakeSessionMemory) Save(_ context.Context, rec intent.MemoryRecord) error {
	if rec.ID == "" {
		rec.ID = "rec-1"
	}
	rec.UpdatedAt = time.Now().UTC()
	s.records[rec.ID] = rec

	return nil
}

func (s *fakeSessionMemory) Search(_ context.Context, q intent.MemoryQuery) ([]intent.MemoryResult, error) {
	var out []intent.MemoryResult
	for _, rec := range s.records {
		if q.Filter["key"] != "" && rec.Key != q.Filter["key"] {
			continue
		}
		out = append(out, intent.MemoryResult{Record: rec, Score: 1})
	}

	return out, nil
}

func (s *fakeSessionMemory) Delete(_ context.Context, id string) error {
	if _, ok := s.records[id]; !ok {
		return errs.ErrRecordNotFound
	}
	delete(s.records, id)

	return nil
}

func (s *fakeSessionMemory) Clear(_ context.Context) error {
	s.records = make(map[string]intent.MemoryRecord)

	return nil
}

var _ intent.Memory = (*fakeMemory)(nil)

func TestManagerCreateGetClose(t *testing.T) {
	mgr := intent.NewSessionManager(newFakeMemory())

	s, err := mgr.Create(context.Background(), intent.SessionWithID("s1"))
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "s1" {
		t.Fatalf("got id %q", s.ID)
	}

	got, err := mgr.Get(context.Background(), "s1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "s1" {
		t.Fatalf("got id %q", got.ID)
	}

	err = mgr.Close(context.Background(), "s1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Get(context.Background(), "s1")
	if !errs.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestManagerDuplicateCreate(t *testing.T) {
	mgr := intent.NewSessionManager(newFakeMemory())

	_, err := mgr.Create(context.Background(), intent.SessionWithID("dup"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Create(context.Background(), intent.SessionWithID("dup"))
	if err == nil {
		t.Fatal("expected duplicate session error")
	}
}

func TestHistoryRoundTrip(t *testing.T) {
	mgr := intent.NewSessionManager(newFakeMemory())

	s, err := mgr.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	msg := intent.Message{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart("hello")}}
	err = intent.AppendHistory(context.Background(), s, msg)
	if err != nil {
		t.Fatal(err)
	}

	history, err := intent.LoadHistory(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Parts[0].Text != "hello" {
		t.Fatalf("unexpected history: %#v", history)
	}
}

func TestManagerNilMemory(t *testing.T) {
	mgr := intent.NewSessionManager(nil)
	_, err := mgr.Create(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errs.ErrUnsupportedOperation) {
		t.Fatalf("got %v", err)
	}
}

func TestManagerConcurrentCreate(t *testing.T) {
	mgr := intent.NewSessionManager(newFakeMemory())
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := mgr.Create(context.Background(), intent.SessionWithID("shared"))
			errCh <- err
		}()
	}

	wg.Wait()
	close(errCh)

	var success, failure int
	for err := range errCh {
		if err == nil {
			success++
		} else {
			failure++
		}
	}
	if success != 1 || failure != 1 {
		t.Fatalf("expected one success and one failure, got success=%d failure=%d", success, failure)
	}
}
