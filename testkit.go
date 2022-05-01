package ronykit

import (
	"context"
	"sync"

	"github.com/clubpay/ronykit/utils"
)

// TestContext is useful for writing end-to-end tests for your Contract handlers.
type TestContext struct {
	ctx        *Context
	handlers   HandlerFuncChain
	inMsg      Message
	inHdr      EnvelopeHdr
	expectFunc func(...*Envelope) error
}

func NewTestContext() *TestContext {
	return &TestContext{}
}

func (testCtx *TestContext) SetHandler(h ...HandlerFunc) *TestContext {
	testCtx.handlers = h

	return testCtx
}

func (testCtx *TestContext) Input(m Message, hdr EnvelopeHdr) *TestContext {
	testCtx.inMsg = m
	testCtx.inHdr = hdr

	return testCtx
}

func (testCtx *TestContext) Expectation(f func(out ...*Envelope) error) *TestContext {
	testCtx.expectFunc = f

	return testCtx
}

func (testCtx *TestContext) Run() error {
	ctx := newContext()
	conn := newTestConn()
	ctx.conn = conn
	ctx.in = newEnvelope(ctx, conn, false).
		SetMsg(testCtx.inMsg).
		SetHdrMap(testCtx.inHdr)
	ctx.ctx = context.Background()
	ctx.wf = func(conn Conn, e *Envelope) error {
		e.DontRelease()
		tc := conn.(*testConn)
		tc.Lock()
		tc.out = append(tc.out, e)
		tc.Unlock()

		return nil
	}
	ctx.handlers = append(ctx.handlers, testCtx.handlers...)
	ctx.Next()

	return testCtx.expectFunc(conn.out...)
}

type testConn struct {
	sync.Mutex

	id       uint64
	clientIP string
	stream   bool
	kv       map[string]string
	out      []*Envelope
}

var _ Conn = (*testConn)(nil)

func newTestConn() *testConn {
	return &testConn{
		id: utils.RandomUint64(0),
	}
}

func (t *testConn) ConnID() uint64 {
	return t.id
}

func (t *testConn) ClientIP() string {
	return t.clientIP
}

func (t *testConn) Write(data []byte) (int, error) {
	return 0, nil
}

func (t *testConn) Stream() bool {
	return t.stream
}

func (t *testConn) Walk(f func(key string, val string) bool) {
	t.Lock()
	defer t.Unlock()

	for k, v := range t.kv {
		if !f(k, v) {
			return
		}
	}
}

func (t *testConn) Get(key string) string {
	t.Lock()
	defer t.Unlock()

	return t.kv[key]
}

func (t *testConn) Set(key string, val string) {
	t.Lock()
	t.kv[key] = val
	t.Unlock()
}
