package kit

import (
	"mime/multipart"
	"sync"

	"github.com/clubpay/ronykit/kit/utils"
)

// TestContext is useful for writing end-to-end tests for your Contract handlers.
type TestContext struct {
	ls         localStore
	handlers   HandlerFuncChain
	inMsg      Message
	inHdr      EnvelopeHdr
	clientIP   string
	expectFunc func(...*Envelope) error
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
	testCtx.expectFunc = f

	return testCtx
}

func (testCtx *TestContext) Run(stream bool) error {
	ctx := newContext(&testCtx.ls)
	conn := newTestConn()
	conn.clientIP = testCtx.clientIP
	conn.stream = stream
	ctx.conn = conn
	ctx.in = newEnvelope(ctx, conn, false)
	ctx.in.
		SetMsg(testCtx.inMsg).
		SetHdrMap(testCtx.inHdr)
	ctx.wf = func(conn Conn, e *Envelope) error {
		e.dontReuse()
		tc := conn.(*testConn) //nolint:forcetypeassert
		tc.Lock()
		tc.out = append(tc.out, e)
		tc.Unlock()

		return nil
	}
	ctx.handlers = append(ctx.handlers, testCtx.handlers...)
	ctx.Next()

	return testCtx.expectFunc(conn.out...)
}

func (testCtx *TestContext) RunREST() error {
	ctx := newContext(&testCtx.ls)
	conn := newTestRESTConn()
	conn.clientIP = testCtx.clientIP
	conn.stream = false
	ctx.conn = conn
	ctx.in = newEnvelope(ctx, conn, false)
	ctx.in.
		SetMsg(testCtx.inMsg).
		SetHdrMap(testCtx.inHdr)
	ctx.wf = func(conn Conn, e *Envelope) error {
		e.dontReuse()
		tc := conn.(*testConn) //nolint:forcetypeassert
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

func (t *testRESTConn) Form() (*multipart.Form, error) {
	panic("not implemented")
}

func (t *testRESTConn) SetStatusCode(code int) {
	t.statusCode = code
}

func (t *testRESTConn) Redirect(code int, url string) {}
