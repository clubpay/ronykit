package kit

import (
	"errors"
	"sync"

	"github.com/clubpay/ronykit/kit/utils"
)

var (
	ErrExpectationsDontMatch = errors.New("expectations don't match")
	ErrUnexpectedEnvelope    = errors.New("unexpected envelope")
)

// TestContext is useful for writing end-to-end tests for your Contract handlers.
type TestContext struct {
	ls       localStore
	handlers HandlerFuncChain
	inMsg    Message
	inHdr    EnvelopeHdr
	clientIP string

	expectFunc   []func(e *Envelope) error
	receiverFunc func(...*Envelope) error
}

func NewTestContext() *TestContext {
	return &TestContext{
		ls: localStore{
			kv: map[string]any{},
		},
	}
}

func (testCtx *TestContext) SetHandler(h ...HandlerFunc) *TestContext {
	testCtx.handlers = h

	return testCtx
}

func (testCtx *TestContext) SetClientIP(ip string) *TestContext {
	testCtx.clientIP = ip

	return testCtx
}

func (testCtx *TestContext) Input(m Message, hdr EnvelopeHdr) *TestContext {
	testCtx.inMsg = m
	testCtx.inHdr = hdr

	return testCtx
}

func (testCtx *TestContext) Receiver(f func(out ...*Envelope) error) *TestContext {
	testCtx.receiverFunc = f

	return testCtx
}

func (testCtx *TestContext) Expect(f func(e *Envelope) error) *TestContext {
	testCtx.expectFunc = append(testCtx.expectFunc, f)

	return testCtx
}

func Expect[IN, OUT, ERR Message](ctx *TestContext, in IN, out *OUT, err *ERR) error {
	return ctx.
		Input(in, EnvelopeHdr{}).
		Expect(func(e *Envelope) error {
			switch e.GetMsg().(type) {
			default:
				return ErrUnexpectedEnvelope
			case OUT:
				*out, _ = e.GetMsg().(OUT)
			case ERR:
				*err, _ = e.GetMsg().(ERR)
			}

			return nil
		}).RunREST()
}

func (testCtx *TestContext) Run(stream bool) error {
	conn := newTestConn()
	conn.clientIP = testCtx.clientIP
	conn.stream = stream

	return testCtx.run(conn)
}

// RunREST simulates a REST request.
func (testCtx *TestContext) RunREST() error {
	conn := newTestRESTConn()
	conn.clientIP = testCtx.clientIP
	conn.stream = false

	return testCtx.run(conn)
}

func (testCtx *TestContext) run(conn Conn) error {
	ctx := newContext(&testCtx.ls)
	ctx.conn = conn
	ctx.in = newEnvelope(ctx, conn, false)
	ctx.in.
		SetMsg(testCtx.inMsg).
		SetHdrMap(testCtx.inHdr)
	ctx.handlers = append(ctx.handlers, testCtx.handlers...)
	ctx.Next()

	var out []*Envelope
	switch conn := conn.(type) {
	default:
		panic("BUG! unknown conn type")
	case *testRESTConn:
		out = conn.out
	case *testConn:
		out = conn.out
	}

	if testCtx.receiverFunc != nil {
		return testCtx.receiverFunc(out...)
	}

	if len(testCtx.expectFunc) > len(out) {
		return ErrExpectationsDontMatch
	}

	for idx, o := range out {
		err := testCtx.expectFunc[idx](o)
		if err != nil {
			return err
		}
	}

	return nil
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
		kv: map[string]string{},
	}
}

func (t *testConn) ConnID() uint64 {
	return t.id
}

func (t *testConn) ClientIP() string {
	return t.clientIP
}

func (t *testConn) Write(_ []byte) (int, error) {
	return 0, nil
}

func (t *testConn) WriteEnvelope(e *Envelope) error {
	e.dontReuse()
	t.Lock()
	t.out = append(t.out, e)
	t.Unlock()

	return nil
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

func (t *testConn) Keys() []string {
	keys := make([]string, 0, len(t.kv))
	for k := range t.kv {
		keys = append(keys, k)
	}

	return keys
}

type testRESTConn struct {
	testConn

	method     string
	path       string
	host       string
	requestURI string
	statusCode int
}

var _ RESTConn = (*testRESTConn)(nil)

func newTestRESTConn() *testRESTConn {
	return &testRESTConn{
		testConn: testConn{
			id: utils.RandomUint64(0),
		},
	}
}

func (t *testRESTConn) WalkQueryParams(f func(key string, val string) bool) {}

func (t *testRESTConn) GetHost() string {
	return t.host
}

func (t *testRESTConn) GetRequestURI() string {
	return t.requestURI
}

func (t *testRESTConn) GetMethod() string {
	return t.method
}

func (t *testRESTConn) GetPath() string {
	return t.path
}

func (t *testRESTConn) SetStatusCode(code int) {
	t.statusCode = code
}

func (t *testRESTConn) Redirect(_ int, _ string) {}
