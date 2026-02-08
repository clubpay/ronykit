package rony

import (
	"context"
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

func TestUnaryCtxHelpers(t *testing.T) {
	state := EMPTY{}
	var nextCalled bool

	handler2 := func(_ *kit.Context) {
		nextCalled = true
	}

	handler1 := func(ctx *kit.Context) {
		u := newUnaryCtx[EMPTY, NOP](ctx, &state, nil)
		if u.State() != state {
			t.Fatalf("unexpected state: %#v", u.State())
		}
		if u.Conn() == nil {
			t.Fatal("expected conn to be set")
		}
		u.SetUserContext(context.WithValue(context.Background(), "k", "v"))
		if u.Context().Value("k") != "v" {
			t.Fatalf("unexpected context value: %v", u.Context().Value("k"))
		}
		_ = u.Route()
		u.Set("a", "b")
		if u.Get("a") != "b" {
			t.Fatalf("unexpected Get value: %v", u.Get("a"))
		}
		if !u.Exists("a") {
			t.Fatal("expected Exists to return true")
		}
		walked := false
		u.Walk(func(key string, val any) bool {
			walked = true

			return false
		})
		if !walked {
			t.Fatal("expected Walk to be called")
		}
		if u.GetInHdr("in") != "header" {
			t.Fatalf("unexpected input header: %s", u.GetInHdr("in"))
		}
		hdrWalked := false
		u.WalkInHdr(func(key string, val string) bool {
			hdrWalked = true

			return false
		})
		if !hdrWalked {
			t.Fatal("expected WalkInHdr to be called")
		}

		if _, ok := u.RESTConn(); !ok {
			t.Fatal("expected REST conn to be available")
		}
		if u.KitCtx() != ctx {
			t.Fatal("expected KitCtx to return the underlying context")
		}

		u.SetOutHdr("x", "1")
		u.SetOutHdrMap(map[string]string{"y": "2"})
		ctx.Out().SetMsg(outMsg{OK: true}).Send()

		u.Next()
	}

	err := kit.NewTestContext().
		Input(&inMsg{ID: 1}, kit.EnvelopeHdr{"in": "header"}).
		SetHandler(handler1, handler2).
		Expect(func(e *kit.Envelope) error {
			if e.GetHdr("x") != "1" || e.GetHdr("y") != "2" {
				t.Fatalf("unexpected output headers: %#v", e)
			}

			return nil
		}).
		RunREST()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !nextCalled {
		t.Fatal("expected Next to call the next handler")
	}
}

func TestUnaryCtxStopExecution(t *testing.T) {
	state := EMPTY{}
	called := false

	handler1 := func(ctx *kit.Context) {
		u := newUnaryCtx[EMPTY, NOP](ctx, &state, nil)
		u.StopExecution()
	}
	handler2 := func(_ *kit.Context) {
		called = true
	}

	err := kit.NewTestContext().
		Input(&inMsg{ID: 1}, kit.EnvelopeHdr{}).
		SetHandler(handler1, handler2).
		RunREST()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("expected StopExecution to prevent next handler")
	}
}
