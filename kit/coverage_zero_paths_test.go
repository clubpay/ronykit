package kit

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeCodec struct {
	decodeErr error
	decoded   bool
}

func (f *fakeCodec) Encode(_ Message, _ io.Writer) error { return nil }
func (f *fakeCodec) Marshal(_ any) ([]byte, error)       { return []byte(`"ok"`), nil }
func (f *fakeCodec) Decode(_ Message, _ io.Reader) error {
	f.decoded = true

	return f.decodeErr
}
func (f *fakeCodec) Unmarshal(_ []byte, _ any) error { return nil }

type walkKV map[string]string

func (w walkKV) Walk(fn func(k, v string) bool) {
	for k, v := range w {
		if !fn(k, v) {
			return
		}
	}
}

type svcWrapFn func(Service) Service

func (f svcWrapFn) Wrap(s Service) Service { return f(s) }

func TestZeroCoverageSimpleHelpers(t *testing.T) {
	t.Run("nop logger methods", func(t *testing.T) {
		var l NOPLogger
		l.Error("x")
		l.Errorf("err=%s", "x")
		l.Debug("x")
		l.Debugf("dbg=%s", "x")
	})

	t.Run("encoding helpers", func(t *testing.T) {
		enc := CustomEncoding("csv")
		if enc.Tag() != "csv" {
			t.Fatalf("unexpected tag: %q", enc.Tag())
		}
	})

	t.Run("service wrappers", func(t *testing.T) {
		base := testService{name: "svc", contracts: []Contract{&testContract{id: "c1"}}}
		w1 := ServiceWrapperFunc(func(s Service) Service {
			return testService{name: s.Name() + "-1", contracts: s.Contracts()}
		})
		w2 := svcWrapFn(func(s Service) Service {
			return testService{name: s.Name() + "-2", contracts: s.Contracts()}
		})

		out := WrapService(base, w1, w2)
		if out.Name() != "svc-1-2" {
			t.Fatalf("unexpected wrapped name: %s", out.Name())
		}
	})
}

func TestZeroCoverageMessageHelpers(t *testing.T) {
	orig := GetMessageCodec()
	defer SetCustomCodec(orig)

	codec := &fakeCodec{}
	SetCustomCodec(codec)
	if GetMessageCodec() != codec {
		t.Fatal("expected custom codec")
	}

	var dst map[string]any
	if err := DecodeMessage(&dst, bytes.NewBufferString(`{"a":1}`)); err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}
	if !codec.decoded {
		t.Fatal("expected codec.Decode to be called")
	}

	raw := RawMessage("abc")
	b, err := raw.Marshal()
	if err != nil || string(b) != "abc" {
		t.Fatalf("Marshal mismatch: %v %q", err, string(b))
	}
	b, err = raw.MarshalJSON()
	if err != nil || string(b) != "abc" {
		t.Fatalf("MarshalJSON mismatch: %v %q", err, string(b))
	}

	var rm RawMessage
	if err := rm.Unmarshal([]byte("x")); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if err := rm.UnmarshalJSON([]byte("y")); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if string(rm) != "xy" {
		t.Fatalf("unexpected raw content: %q", string(rm))
	}

	var mf MultipartFormMessage
	form := &multipart.Form{}
	mf.SetForm(form)
	if mf.GetForm() != form {
		t.Fatal("expected multipart form to be preserved")
	}
}

func TestZeroCoverageContextAndEnvelopeHelpers(t *testing.T) {
	ls := &localStore{kv: map[string]any{}}
	ctx := newContext(ls)
	conn := newTestConn()
	ctx.conn = conn
	ctx.in = newEnvelope(ctx, conn, false)
	ctx.rawData = []byte("payload")
	ctx.setRoute("route").setServiceName("svc").setContractID("cid")
	ctx.sb = &southBridge{}

	if got := ctx.GetStatusText(); got != "OK" {
		t.Fatalf("unexpected status text: %q", got)
	}
	ctx.SetStatusCode(404)
	if got := ctx.GetStatusText(); got != "Not Found" {
		t.Fatalf("unexpected status text: %q", got)
	}
	if string(ctx.InputRawData()) != "payload" {
		t.Fatalf("unexpected raw data: %q", string(ctx.InputRawData()))
	}

	rc := newTestRESTConn()
	ctx.conn = rc
	if ctx.RESTConn() != rc {
		t.Fatal("expected RESTConn cast to return concrete conn")
	}

	ctx.conn = conn
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic on RESTConn cast for non-rest conn")
			}
		}()
		_ = ctx.RESTConn()
	}()

	ctx.Set("bytes", []byte("ok"))
	if string(ctx.GetBytes("bytes", nil)) != "ok" {
		t.Fatalf("unexpected bytes value: %q", string(ctx.GetBytes("bytes", nil)))
	}
	if string(ctx.GetBytes("missing", []byte("def"))) != "def" {
		t.Fatalf("unexpected default bytes value: %q", string(ctx.GetBytes("missing", []byte("def"))))
	}

	count := 0
	ctx.Set("a", 1).Set("b", 2)
	ctx.Walk(func(_ string, _ any) bool {
		count++

		return false
	})
	if count != 1 {
		t.Fatalf("expected early stop in Walk, got: %d", count)
	}

	out := ctx.Out()
	out.SetHdrWalker(walkKV{"k1": "v1", "k2": "v2"})
	seen := 0
	out.WalkHdr(func(_, _ string) bool {
		seen++

		return false
	})
	if seen != 1 {
		t.Fatalf("expected WalkHdr early stop, got: %d", seen)
	}

	reply := out.SetID("id-1").Reply()
	if reply.GetID() != "id-1" || !reply.IsOutgoing() {
		t.Fatalf("unexpected reply envelope state: id=%q outgoing=%v", reply.GetID(), reply.IsOutgoing())
	}
	if out.SizeHint() != CodecDefaultBufferSize {
		t.Fatalf("unexpected default size hint: %d", out.SizeHint())
	}
	if out.SetSizeHint(2048).SizeHint() != 2048 {
		t.Fatalf("unexpected custom size hint: %d", out.SizeHint())
	}
}

func TestZeroCoverageEdgeHelpers(t *testing.T) {
	origForkVal, hadFork := os.LookupEnv(envForkChildKey)
	if err := os.Setenv(envForkChildKey, "7"); err != nil {
		t.Fatalf("setenv failed: %v", err)
	}
	if childID() != 7 {
		t.Fatalf("unexpected child id: %d", childID())
	}
	if hadFork {
		_ = os.Setenv(envForkChildKey, origForkVal)
	} else {
		_ = os.Unsetenv(envForkChildKey)
	}

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "UNKNOWN"}
	for _, m := range methods {
		if got := httpMethodColor(m).Sprint(m); got == "" {
			t.Fatalf("unexpected empty method color output for %s", m)
		}
	}

	for _, code := range []int{200, 302, 404, 503} {
		icon, clr := statusIndicator(code)
		if icon == "" || clr.Sprintf("%d", code) == "" {
			t.Fatalf("unexpected indicator/color for status %d", code)
		}
	}

	ls := &localStore{kv: map[string]any{}}
	ctx := newContext(ls)
	conn := newTestConn()
	ctx.conn = conn
	ctx.in = newEnvelope(ctx, conn, false)
	ctx.setRoute("DoThing")
	ctx.setServiceName("svc")
	ctx.SetStatusCode(500)

	var buf bytes.Buffer
	writeEndpointLog(&endpointLog{w: &buf}, ctx, 2*time.Millisecond)
	if !strings.Contains(buf.String(), "svc") {
		t.Fatalf("expected service in endpoint log, got: %q", buf.String())
	}
}

func TestZeroCoverageContractLookupAndTestRESTNoops(t *testing.T) {
	_, err := resolveContract(map[string]Contract{}, "", "cid")
	if !errors.Is(err, ErrContractNotFound) {
		t.Fatalf("expected ErrContractNotFound for invalid args, got: %v", err)
	}

	want := &testContract{id: "cid"}
	got, err := resolveContract(map[string]Contract{contractLookupKey("svc", "cid"): want}, "svc", "cid")
	if err != nil {
		t.Fatalf("resolveContract unexpected error: %v", err)
	}
	if got != want {
		t.Fatal("expected resolved contract")
	}

	rc := newTestRESTConn()
	called := false
	rc.WalkQueryParams(func(_, _ string) bool {
		called = true

		return false
	})
	if called {
		t.Fatal("WalkQueryParams should be a no-op in test REST conn")
	}
	rc.Redirect(301, "https://example.com")
}

func TestZeroCoverageShutdownPreforkParentNoop(t *testing.T) {
	origForkVal, hadFork := os.LookupEnv(envForkChildKey)
	_ = os.Unsetenv(envForkChildKey) // parent process when prefork is enabled
	defer func() {
		if hadFork {
			_ = os.Setenv(envForkChildKey, origForkVal)
		} else {
			_ = os.Unsetenv(envForkChildKey)
		}
	}()

	gw := &testGateway{}
	s := NewServer(WithGateway(gw), WithPrefork())
	s.Shutdown(context.Background())
	if gw.shutdownCalls != 0 {
		t.Fatalf("expected no shutdown call in prefork parent path, got: %d", gw.shutdownCalls)
	}
}

func TestEdgeCaseEnvelopeSendPanics(t *testing.T) {
	t.Run("panic on nil conn", func(t *testing.T) {
		e := &Envelope{outgoing: true}
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic when sending with nil conn")
			}
		}()
		e.Send()
	})

	t.Run("panic on incoming envelope", func(t *testing.T) {
		ctx := newContext(&localStore{kv: map[string]any{}})
		conn := newTestConn()
		e := newEnvelope(ctx, conn, false)
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic when sending incoming envelope")
			}
		}()
		e.Send()
	})
}

func TestEdgeCaseNorthBridgeNilConnDelegate(t *testing.T) {
	b := &northBridge{}
	b.OnOpen(newTestConn())
	b.OnClose(1)
}

func TestEdgeCaseSouthBridgeUnknownSessionCarrier(t *testing.T) {
	ls := &localStore{kv: map[string]any{}}
	errCalls := 0
	sb := &southBridge{
		ctxPool:      ctxPool{ls: ls},
		wg:           &sync.WaitGroup{},
		eh:           func(_ *Context, _ error) { errCalls++ },
		inProgress:   map[string]*clusterConn{},
		msgFactories: map[string]MessageFactoryFunc{},
	}

	c := newEnvelopeCarrier(outgoingCarrier, "unknown-session", "origin", "target")
	var b bytes.Buffer
	if err := defaultMessageCodec.Encode(c, &b); err != nil {
		t.Fatalf("failed to encode carrier: %v", err)
	}

	sb.OnMessage(b.Bytes())
	if errCalls != 0 {
		t.Fatalf("unexpected error callback calls: %d", errCalls)
	}
}

func TestEdgeCaseForwarderHandlerBranching(t *testing.T) {
	newCtx := func() *Context {
		ctx := newContext(&localStore{kv: map[string]any{}})
		conn := newTestConn()
		ctx.conn = conn
		ctx.in = newEnvelope(ctx, conn, false)

		return ctx
	}

	t.Run("already forwarded request returns quickly", func(t *testing.T) {
		sb := &southBridge{id: "node-1"}
		h := sb.genForwarderHandler(func(*LimitedContext) (string, error) {
			t.Fatal("selector should not be called for forwarded contexts")

			return "", nil
		})

		ctx := newCtx()
		ctx.forwarded = true
		h(ctx)
		if ctx.HasError() {
			t.Fatal("unexpected error on forwarded context path")
		}
	})

	t.Run("selector error is recorded", func(t *testing.T) {
		sb := &southBridge{id: "node-1"}
		h := sb.genForwarderHandler(func(*LimitedContext) (string, error) {
			return "", errors.New("selector-failed")
		})

		ctx := newCtx()
		ctx.sb = sb
		h(ctx)
		if !ctx.HasError() {
			t.Fatal("expected selector error to be recorded")
		}
	})

	t.Run("empty or self target skips forwarding", func(t *testing.T) {
		sb := &southBridge{id: "node-1"}
		hEmpty := sb.genForwarderHandler(func(*LimitedContext) (string, error) {
			return "", nil
		})
		hSelf := sb.genForwarderHandler(func(*LimitedContext) (string, error) {
			return "node-1", nil
		})

		ctx1 := newCtx()
		ctx1.sb = sb
		hEmpty(ctx1)
		if ctx1.handlerIndex == abortIndex {
			t.Fatal("expected no execution abort for empty target")
		}

		ctx2 := newCtx()
		ctx2.sb = sb
		hSelf(ctx2)
		if ctx2.handlerIndex == abortIndex {
			t.Fatal("expected no execution abort for self target")
		}
	})
}

func TestEdgeCaseCastRawMessageInvalidPayload(t *testing.T) {
	cases := []struct {
		name    string
		payload RawMessage
	}{
		{"not json", RawMessage("not-json")},
		{"empty", RawMessage("")},
		{"null bytes", RawMessage("\x00\x00\x00")},
		{"truncated json", RawMessage(`{"a":`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CastRawMessage[map[string]any](tc.payload)
			if err == nil {
				t.Fatalf("expected cast to fail for %q", tc.name)
			}
		})
	}
}
