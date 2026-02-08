package rony

import (
	"testing"

	"github.com/clubpay/ronykit/kit"
)

type testAction int

type testState struct {
	value    int
	locked   int
	unlocked int
}

func (s *testState) Name() string {
	return "test"
}

func (s *testState) Reduce(a testAction) error {
	s.value += int(a)

	return nil
}

func (s *testState) Lock() {
	s.locked++
}

func (s *testState) Unlock() {
	s.unlocked++
}

type inMsg struct {
	ID int `json:"id"`
}

type outMsg struct {
	OK bool `json:"ok"`
}

func TestBaseCtxReduceStateLocking(t *testing.T) {
	state := &testState{}
	ctx := &BaseCtx[*testState, testAction]{
		s:  state,
		sl: state,
	}

	called := false
	err := ctx.ReduceState(2, func(s *testState, err error) error {
		called = true
		if s.value != 2 {
			t.Fatalf("unexpected state value in callback: %d", s.value)
		}

		return err
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected callback to be called")
	}
	if state.locked != 1 || state.unlocked != 1 {
		t.Fatalf("unexpected lock counts: %d/%d", state.locked, state.unlocked)
	}

	stateNoLock := &testState{}
	ctxNoLock := &BaseCtx[*testState, testAction]{
		s: stateNoLock,
	}
	_ = ctxNoLock.ReduceState(3, nil)
	if stateNoLock.locked != 0 || stateNoLock.unlocked != 0 {
		t.Fatalf("unexpected lock counts without locker: %d/%d", stateNoLock.locked, stateNoLock.unlocked)
	}
	if stateNoLock.value != 3 {
		t.Fatalf("unexpected state value without locker: %d", stateNoLock.value)
	}
}

func TestStreamCtxPushHeaders(t *testing.T) {
	state := EMPTY{}

	handler := func(ctx *kit.Context) {
		streamCtx := newStreamCtx[EMPTY, NOP, outMsg](ctx, &state, nil)
		streamCtx.Push(outMsg{OK: true},
			WithHdr("x", "1"),
			WithHdrMap(map[string]string{"y": "2"}),
		)
	}

	err := kit.NewTestContext().
		Input(&inMsg{ID: 1}, kit.EnvelopeHdr{}).
		SetHandler(handler).
		Expect(func(e *kit.Envelope) error {
			if got := e.GetHdr("x"); got != "1" {
				t.Fatalf("unexpected header x: %s", got)
			}
			if got := e.GetHdr("y"); got != "2" {
				t.Fatalf("unexpected header y: %s", got)
			}
			if _, ok := e.GetMsg().(outMsg); !ok {
				t.Fatalf("unexpected message type: %T", e.GetMsg())
			}

			return nil
		}).
		Run(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
