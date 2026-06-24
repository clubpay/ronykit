package intent_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"
)

type seqLLM struct {
	info      intent.Model
	responses []intent.Response
	calls     int
}

func (m *seqLLM) Model() intent.Model { return m.info }

func (m *seqLLM) Generate(_ context.Context, _ intent.Request) (intent.Response, error) {
	if m.calls >= len(m.responses) {
		return intent.Response{Content: "done"}, nil
	}
	resp := m.responses[m.calls]
	m.calls++

	return resp, nil
}

func (m *seqLLM) Stream(context.Context, intent.Request) (intent.Stream, error) {
	return nil, errors.New("not implemented")
}

type fakeStatic struct {
	entries []intent.Entry
}

func (f fakeStatic) List(_ context.Context, _ intent.Filter) ([]intent.Entry, error) {
	return f.entries, nil
}

func (f fakeStatic) Get(_ context.Context, id string) (intent.Entry, error) {
	for _, entry := range f.entries {
		if entry.ID == id {
			return entry, nil
		}
	}

	return intent.Entry{}, errs.ErrKnowledgeNotFound
}

type mapRetriever map[string][]intent.Entry

func (m mapRetriever) Retrieve(_ context.Context, q intent.RetrieveQuery) ([]intent.Entry, error) {
	return m[q.Text], nil
}

type runtimeFakeMemory struct {
	sessions map[string]intent.SessionMemory
}

type runtimeFakeSessionMemory struct {
	id      string
	records map[string]intent.MemoryRecord
}

func newRuntimeFakeMemory() *runtimeFakeMemory {
	return &runtimeFakeMemory{sessions: make(map[string]intent.SessionMemory)}
}

func (m *runtimeFakeMemory) ForSession(id string) intent.SessionMemory {
	s, ok := m.sessions[id]
	if !ok {
		s = &runtimeFakeSessionMemory{id: id, records: make(map[string]intent.MemoryRecord)}
		m.sessions[id] = s
	}

	return s
}

func (s *runtimeFakeSessionMemory) SessionID() string { return s.id }

func (s *runtimeFakeSessionMemory) Save(_ context.Context, rec intent.MemoryRecord) error {
	if rec.ID == "" {
		rec.ID = "1"
	}
	s.records[rec.ID] = rec

	return nil
}

func (s *runtimeFakeSessionMemory) Search(_ context.Context, q intent.MemoryQuery) ([]intent.MemoryResult, error) {
	var out []intent.MemoryResult
	for _, rec := range s.records {
		if q.Filter["key"] != "" && rec.Key != q.Filter["key"] {
			continue
		}
		out = append(out, intent.MemoryResult{Record: rec, Score: 1})
	}

	return out, nil
}

func (s *runtimeFakeSessionMemory) Delete(context.Context, string) error { return nil }
func (s *runtimeFakeSessionMemory) Clear(context.Context) error {
	s.records = make(map[string]intent.MemoryRecord)

	return nil
}

func TestRunTurnHappyPath(t *testing.T) {
	mem := newRuntimeFakeMemory()
	sessMgr := intent.NewSessionManager(mem)
	s, err := sessMgr.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	reg := intent.NewToolRegistry()
	err = reg.Register(intent.LocalToolFunc{
		Def: intent.ToolDefinition{Name: "echo", Description: "echo"},
		Fn: func(_ context.Context, args json.RawMessage) (intent.Message, error) {
			return intent.Message{Role: intent.RoleTool, Parts: []intent.Part{intent.TextPart(string(args))}}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	model := &seqLLM{
		info:      intent.Model{ID: "mock"},
		responses: []intent.Response{{Content: "final answer"}},
	}
	pool, err := intent.NewLLMPool([]intent.LLM{model}, intent.SelectorFunc(intent.SelectFirst))
	if err != nil {
		t.Fatal(err)
	}

	agent := intent.New(
		intent.WithStaticKnowledge(fakeStatic{entries: []intent.Entry{{
			Kind: intent.KindPrompt, Name: "system", Content: "You are helpful",
		}}}),
		intent.WithLLMPool(pool),
		intent.WithTools(reg),
		intent.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))),
	)

	result, err := agent.RunTurn(context.Background(), intent.TurnInput{
		Session:     s,
		UserMessage: intent.Message{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart("hello")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Response.Content != "final answer" {
		t.Fatalf("got %q", result.Response.Content)
	}

	history, err := intent.LoadHistory(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history messages, got %d", len(history))
	}
}

func TestRunTurnToolLoop(t *testing.T) {
	mem := newRuntimeFakeMemory()
	sessMgr := intent.NewSessionManager(mem)
	s, err := sessMgr.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	reg := intent.NewToolRegistry()
	err = reg.Register(intent.LocalToolFunc{
		Def: intent.ToolDefinition{Name: "echo"},
		Fn: func(_ context.Context, args json.RawMessage) (intent.Message, error) {
			return intent.Message{Role: intent.RoleTool, Parts: []intent.Part{intent.TextPart("tool:" + string(args))}}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	model := &seqLLM{
		info: intent.Model{ID: "mock"},
		responses: []intent.Response{
			{ToolCalls: []intent.ToolCall{{ID: "1", Name: "echo", Arguments: `{"x":1}`}}},
			{Content: "done"},
		},
	}
	pool, err := intent.NewLLMPool([]intent.LLM{model}, intent.SelectorFunc(intent.SelectFirst))
	if err != nil {
		t.Fatal(err)
	}

	agent := intent.New(
		intent.WithLLMPool(pool),
		intent.WithTools(reg),
	)

	result, err := agent.RunTurn(context.Background(), intent.TurnInput{
		Session:     s,
		UserMessage: intent.Message{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart("run tool")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Response.Content != "done" {
		t.Fatalf("got %q", result.Response.Content)
	}
	if model.calls != 2 {
		t.Fatalf("expected 2 llm calls, got %d", model.calls)
	}
}

func TestRunTurnMaxToolIterations(t *testing.T) {
	mem := newRuntimeFakeMemory()
	s, err := intent.NewSessionManager(mem).Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	reg := intent.NewToolRegistry()
	err = reg.Register(intent.LocalToolFunc{
		Def: intent.ToolDefinition{Name: "loop"},
		Fn: func(_ context.Context, _ json.RawMessage) (intent.Message, error) {
			return intent.Message{Role: intent.RoleTool, Parts: []intent.Part{intent.TextPart("again")}}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	model := &seqLLM{
		info: intent.Model{ID: "mock"},
		responses: []intent.Response{
			{ToolCalls: []intent.ToolCall{{ID: "1", Name: "loop", Arguments: `{}`}}},
			{ToolCalls: []intent.ToolCall{{ID: "2", Name: "loop", Arguments: `{}`}}},
		},
	}
	pool, err := intent.NewLLMPool([]intent.LLM{model}, intent.SelectorFunc(intent.SelectFirst))
	if err != nil {
		t.Fatal(err)
	}

	agent := intent.New(
		intent.WithLLMPool(pool),
		intent.WithTools(reg),
		intent.WithMaxToolIterations(2),
	)

	_, err = agent.RunTurn(context.Background(), intent.TurnInput{
		Session:     s,
		UserMessage: intent.Message{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart("loop")}},
	})
	if !errors.Is(err, errs.ErrMaxToolIterations) {
		t.Fatalf("expected max iterations error, got %v", err)
	}
}

func TestRunTurnWithRAG(t *testing.T) {
	mem := newRuntimeFakeMemory()
	s, err := intent.NewSessionManager(mem).Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	model := &seqLLM{
		info:      intent.Model{ID: "mock"},
		responses: []intent.Response{{Content: "ok"}},
	}
	pool, err := intent.NewLLMPool([]intent.LLM{model}, intent.SelectorFunc(intent.SelectFirst))
	if err != nil {
		t.Fatal(err)
	}

	agent := intent.New(
		intent.WithRetriever(mapRetriever{"billing": {{Content: "refund policy", Origin: intent.OriginDynamic, Name: "policy"}}}),
		intent.WithLLMPool(pool),
	)

	_, err = agent.RunTurn(context.Background(), intent.TurnInput{
		Session:       s,
		UserMessage:   intent.Message{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart("question")}},
		RetrieveQuery: "billing",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunTurnNilSession(t *testing.T) {
	pool, err := intent.NewLLMPool([]intent.LLM{&seqLLM{info: intent.Model{ID: "mock"}}}, intent.SelectorFunc(intent.SelectFirst))
	if err != nil {
		t.Fatal(err)
	}

	agent := intent.New(intent.WithLLMPool(pool))

	_, err = agent.RunTurn(context.Background(), intent.TurnInput{
		UserMessage: intent.Message{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart("x")}},
	})
	if !errs.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestRunTurnRequiresPool(t *testing.T) {
	agent := intent.New()

	_, err := agent.RunTurn(context.Background(), intent.TurnInput{
		UserMessage: intent.Message{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart("x")}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
