package services

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/clubpay/ronykit/kit"
)

func TestContextMWSetsDeadline(t *testing.T) {
	var (
		gotDeadline bool
		seenCtx     context.Context
	)

	ctx := kit.NewTestContext().
		Input(&EchoRequest{}, kit.EnvelopeHdr{}).
		SetHandler(
			contextMW(time.Millisecond*50),
			func(ctx *kit.Context) {
				seenCtx = ctx.Context()
				_, gotDeadline = seenCtx.Deadline()
			},
		)

	if err := ctx.RunREST(); err != nil {
		t.Fatal(err)
	}
	if !gotDeadline {
		t.Fatal("expected deadline in user context")
	}

	select {
	case <-seenCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("expected user context to be canceled")
	}
}

func TestTestClusterStoreLocal(t *testing.T) {
	store := testClusterStore{
		sharedKV: map[string]string{},
	}

	ctx := kit.NewTestContext().
		Input(&SetRequest{Key: "k", Value: "v"}, kit.EnvelopeHdr{}).
		SetHandler(func(ctx *kit.Context) {
			setDummyCluster(t, ctx)
			if err := store.Set(ctx, "k", "v", 0); err != nil {
				t.Fatalf("unexpected set error: %v", err)
			}
			val, err := store.Get(ctx.Limited(), "k")
			if err != nil || val != "v" {
				t.Fatalf("unexpected get result: %v %s", err, val)
			}
			if _, err := store.Get(ctx.Limited(), "missing"); err == nil {
				t.Fatal("expected missing key error")
			}
		})

	if err := ctx.RunREST(); err != nil {
		t.Fatal(err)
	}
}

func TestKeyValueCoordinator(t *testing.T) {
	sharedKV.sharedKV = map[string]string{
		"key1": "node-a",
		"key2": "node-b",
	}

	ctx := kit.NewTestContext().
		Input(&SetRequest{Key: "key1"}, kit.EnvelopeHdr{}).
		SetHandler(func(ctx *kit.Context) {
			setDummyCluster(t, ctx)
			val, err := keyValueCoordinator(ctx.Limited())
			if err != nil || val != "node-a" {
				t.Fatalf("unexpected coordinator result: %v %s", err, val)
			}
		})
	if err := ctx.RunREST(); err != nil {
		t.Fatal(err)
	}

	ctx = kit.NewTestContext().
		Input(&GetRequest{Key: "key2"}, kit.EnvelopeHdr{}).
		SetHandler(func(ctx *kit.Context) {
			setDummyCluster(t, ctx)
			val, err := keyValueCoordinator(ctx.Limited())
			if err != nil || val != "node-b" {
				t.Fatalf("unexpected coordinator result: %v %s", err, val)
			}
		})
	if err := ctx.RunREST(); err != nil {
		t.Fatal(err)
	}
}

func TestEchoServiceHandlers(t *testing.T) {
	svc := EchoService.Build()

	var echoContract kit.Contract
	var closeContract kit.Contract
	for _, c := range svc.Contracts() {
		switch c.Input().(type) {
		case *EchoRequest:
			echoContract = c
		case *CloseRequest:
			closeContract = c
		}
	}

	if echoContract == nil || closeContract == nil {
		t.Fatal("expected echo and close contracts")
	}

	var out EchoResponse
	ctx := kit.NewTestContext().
		Input(&EchoRequest{Input: "hello"}, kit.EnvelopeHdr{}).
		SetHandler(echoContract.Handlers()...).
		Expect(func(e *kit.Envelope) error {
			msg := e.GetMsg().(*EchoResponse) //nolint:forcetypeassert
			out = *msg

			return nil
		})
	if err := ctx.RunREST(); err != nil {
		t.Fatal(err)
	}
	if out.Output != "hello" {
		t.Fatalf("unexpected echo output: %s", out.Output)
	}

	var closeOut CloseResponse
	var wrapper *rpcConnWrapper
	closeHandlers := append([]kit.HandlerFunc{
		func(ctx *kit.Context) {
			wrapper = &rpcConnWrapper{Conn: ctx.Conn()}
			setConn(t, ctx, wrapper)
			ctx.Next()
		},
	}, closeContract.Handlers()...)
	ctx = kit.NewTestContext().
		Input(&CloseRequest{}, kit.EnvelopeHdr{}).
		SetHandler(closeHandlers...).
		Expect(func(e *kit.Envelope) error {
			msg := e.GetMsg().(*CloseResponse) //nolint:forcetypeassert
			closeOut = *msg

			return nil
		})
	if err := ctx.RunREST(); err != nil {
		t.Fatal(err)
	}
	if !closeOut.Close {
		t.Fatalf("unexpected close response: %+v", closeOut)
	}
	if wrapper == nil || !wrapper.closed {
		t.Fatalf("expected rpc close to be called")
	}
}

func TestEchoRawServiceHandler(t *testing.T) {
	svc := EchoRawService.Build()

	var rawContract kit.Contract
	for _, c := range svc.Contracts() {
		if _, ok := c.Input().(*kit.RawMessage); ok {
			rawContract = c
			break
		}
	}
	if rawContract == nil {
		t.Fatal("expected raw contract")
	}

	var out kit.RawMessage
	ctx := kit.NewTestContext().
		Input(kit.RawMessage(`{}`), kit.EnvelopeHdr{}).
		SetHandler(rawContract.Handlers()...).
		Expect(func(e *kit.Envelope) error {
			out = e.GetMsg().(kit.RawMessage) //nolint:forcetypeassert

			return nil
		})
	if err := ctx.RunREST(); err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("expected raw response body")
	}
}

func TestKeyValueServiceHandlers(t *testing.T) {
	svc := SimpleKeyValueService.Build()

	var setContract kit.Contract
	var getContract kit.Contract
	for _, c := range svc.Contracts() {
		switch c.Input().(type) {
		case *SetRequest:
			setContract = c
		case *GetRequest:
			getContract = c
		}
	}
	if setContract == nil || getContract == nil {
		t.Fatal("expected set/get contracts")
	}

	sharedKV.sharedKV = map[string]string{}

	var setOut KeyValue
	setHandlers := append([]kit.HandlerFunc{
		func(ctx *kit.Context) {
			setDummyCluster(t, ctx)
			ctx.Next()
		},
	}, setContract.Handlers()...)
	ctx := kit.NewTestContext().
		Input(&SetRequest{Key: "k1", Value: "v1"}, kit.EnvelopeHdr{}).
		SetHandler(setHandlers...).
		Expect(func(e *kit.Envelope) error {
			msg := e.GetMsg().(*KeyValue) //nolint:forcetypeassert
			setOut = *msg

			return nil
		})
	if err := ctx.RunREST(); err != nil {
		t.Fatal(err)
	}
	if setOut.Key != "k1" || setOut.Value != "v1" {
		t.Fatalf("unexpected set response: %+v", setOut)
	}

	var getOut KeyValue
	getHandlers := append([]kit.HandlerFunc{
		func(ctx *kit.Context) {
			setDummyCluster(t, ctx)
			ensureConnKV(t, ctx)
			ctx.LocalStore().Set("k1", "v1")
			ctx.Next()
		},
	}, getContract.Handlers()...)
	ctx = kit.NewTestContext().
		Input(&GetRequest{Key: "k1"}, kit.EnvelopeHdr{"Envelope-Hdr-In": "in"}).
		SetHandler(getHandlers...).
		Expect(func(e *kit.Envelope) error {
			msg := e.GetMsg().(*KeyValue) //nolint:forcetypeassert
			getOut = *msg

			return nil
		})
	if err := ctx.RunREST(); err != nil {
		t.Fatal(err)
	}
	if getOut.Key != "k1" || getOut.Value != "v1" {
		t.Fatalf("unexpected get response: %+v", getOut)
	}

	ctx = kit.NewTestContext().
		Input(&GetRequest{Key: "missing"}, kit.EnvelopeHdr{}).
		SetHandler(func(ctx *kit.Context) {
			setDummyCluster(t, ctx)
			ensureConnKV(t, ctx)
			handler := getContract.Handlers()[len(getContract.Handlers())-1]
			handler(ctx)
			if ctx.GetStatusCode() != http.StatusNotFound {
				t.Fatalf("unexpected status code: %d", ctx.GetStatusCode())
			}
		})
	if err := ctx.RunREST(); err != nil {
		t.Fatal(err)
	}
}

type dummyCluster struct{}

func (dummyCluster) Start(context.Context) error { return nil }
func (dummyCluster) Shutdown(context.Context) error {
	return nil
}
func (dummyCluster) Subscribe(string, kit.ClusterDelegate) {}
func (dummyCluster) Publish(string, []byte) error          { return nil }
func (dummyCluster) Subscribers() ([]string, error)        { return []string{}, nil }

func setDummyCluster(t *testing.T, ctx *kit.Context) {
	t.Helper()

	val := reflect.ValueOf(ctx).Elem()
	sbField := val.FieldByName("sb")
	if !sbField.IsValid() {
		t.Fatal("missing sb field")
	}

	sbPtr := reflect.New(sbField.Type().Elem())
	cbField := sbPtr.Elem().FieldByName("cb")
	if !cbField.IsValid() {
		t.Fatal("missing cb field")
	}

	setUnexportedField(cbField, reflect.ValueOf(dummyCluster{}))
	setUnexportedField(sbField, sbPtr)
}

func setUnexportedField(field, value reflect.Value) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(value)
}

func ensureConnKV(t *testing.T, ctx *kit.Context) {
	t.Helper()

	connVal := reflect.ValueOf(ctx.Conn()).Elem()
	kvField := connVal.FieldByName("kv")
	if !kvField.IsValid() {
		t.Fatal("missing kv field")
	}
	if kvField.IsNil() {
		setUnexportedField(kvField, reflect.ValueOf(map[string]string{}))
	}
}

type rpcConnWrapper struct {
	kit.Conn
	closed bool
}

func (r *rpcConnWrapper) Write(p []byte) (int, error) {
	return len(p), nil
}

func (r *rpcConnWrapper) Close() {
	r.closed = true
}

func setConn(t *testing.T, ctx *kit.Context, conn kit.Conn) {
	t.Helper()

	val := reflect.ValueOf(ctx).Elem()
	connField := val.FieldByName("conn")
	if !connField.IsValid() {
		t.Fatal("missing conn field")
	}
	setUnexportedField(connField, reflect.ValueOf(conn))
}
