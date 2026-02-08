package kit

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type bridgeIn struct {
	Name string `json:"name"`
}

type bridgeOut struct {
	OK bool `json:"ok"`
}

func TestNorthBridgeOnOpenClose(t *testing.T) {
	cd := &testConnDelegate{}
	b := &northBridge{cd: cd}
	conn := newTestConn()

	b.OnOpen(conn)
	b.OnClose(conn.ConnID())

	if len(cd.opened) != 1 {
		t.Fatalf("unexpected open calls: %d", len(cd.opened))
	}
	if len(cd.closed) != 1 {
		t.Fatalf("unexpected close calls: %d", len(cd.closed))
	}
}

func TestNorthBridgeOnMessagePaths(t *testing.T) {
	ls := &localStore{kv: map[string]any{}}
	gw := &testGateway{}
	var handled bool
	contract := &testContract{
		id: "c1",
		handlers: []HandlerFunc{
			func(ctx *Context) {
				handled = true
				ctx.Out().SetMsg(&bridgeOut{OK: true}).Send()
			},
		},
	}

	var gotErr error
	b := &northBridge{
		ctxPool: ctxPool{ls: ls},
		wg:      &sync.WaitGroup{},
		eh:      func(_ *Context, err error) { gotErr = err },
		c:       map[string]Contract{"c1": contract},
		gw:      gw,
	}

	gw.dispatchFn = func(_ *Context, _ []byte) (ExecuteArg, error) {
		return ExecuteArg{}, ErrPreflight
	}
	b.OnMessage(newTestConn(), []byte("preflight"))
	if gotErr != nil {
		t.Fatalf("unexpected error on preflight: %v", gotErr)
	}
	if handled {
		t.Fatal("handler should not run on preflight")
	}

	gw.dispatchFn = func(_ *Context, _ []byte) (ExecuteArg, error) {
		return ExecuteArg{}, errors.New("boom")
	}
	b.OnMessage(newTestConn(), []byte("bad"))
	if gotErr == nil || !errors.Is(gotErr, ErrDispatchFailed) {
		t.Fatalf("expected ErrDispatchFailed, got: %v", gotErr)
	}

	gotErr = nil
	gw.dispatchFn = func(_ *Context, _ []byte) (ExecuteArg, error) {
		return ExecuteArg{ContractID: "c1"}, nil
	}
	b.OnMessage(newTestConn(), []byte("ok"))
	if gotErr != nil {
		t.Fatalf("unexpected error: %v", gotErr)
	}
	if !handled {
		t.Fatal("expected handler to run")
	}
}

func TestSouthBridgeCreateSenderConn(t *testing.T) {
	cluster := &testCluster{}
	sb := &southBridge{
		cb:         cluster,
		inProgress: map[string]*clusterConn{},
	}

	carrier := newEnvelopeCarrier(outgoingCarrier, "s1", "origin", "target")
	called := false
	conn := sb.createSenderConn(carrier, 0, func(_ *envelopeCarrier) {
		called = true
	})

	conn.carrierChan <- &envelopeCarrier{Kind: outgoingCarrier, Data: &carrierData{}}
	conn.carrierChan <- &envelopeCarrier{Kind: eofCarrier}

	deadline := time.After(200 * time.Millisecond)
	for {
		if called && sb.getConn("s1") == nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for sender conn cleanup")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestSouthBridgeOnMessage(t *testing.T) {
	cluster := &testCluster{}
	ls := &localStore{kv: map[string]any{}}
	errCalls := 0

	sb := &southBridge{
		ctxPool:      ctxPool{ls: ls},
		wg:           &sync.WaitGroup{},
		eh:           func(_ *Context, _ error) { errCalls++ },
		cb:           cluster,
		inProgress:   map[string]*clusterConn{},
		msgFactories: map[string]MessageFactoryFunc{},
	}

	sb.OnMessage([]byte("bad-json"))
	if errCalls == 0 {
		t.Fatal("expected error handler to be called")
	}

	carrier := newEnvelopeCarrier(outgoingCarrier, "sid", "origin", "target")
	called := false
	conn := sb.createSenderConn(carrier, 0, func(_ *envelopeCarrier) {
		called = true
	})

	var buf bytes.Buffer
	_ = defaultMessageCodec.Encode(&envelopeCarrier{Kind: outgoingCarrier, SessionID: "sid"}, &buf)
	sb.OnMessage(buf.Bytes())

	deadline := time.After(200 * time.Millisecond)
	for {
		if called {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for callback")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	conn.carrierChan <- &envelopeCarrier{Kind: eofCarrier}
}

func TestSouthBridgeSendMessageError(t *testing.T) {
	cluster := &testCluster{publishErr: errors.New("boom")}
	sb := &southBridge{
		cb:         cluster,
		inProgress: map[string]*clusterConn{},
	}

	carrier := newEnvelopeCarrier(outgoingCarrier, "s1", "origin", "target")
	sb.inProgress["s1"] = &clusterConn{}

	err := sb.sendMessage(carrier)
	if err == nil {
		t.Fatal("expected sendMessage error")
	}
	if sb.getConn("s1") != nil {
		t.Fatal("expected in-progress entry to be removed")
	}
}

func TestSouthBridgeOnIncomingMessage(t *testing.T) {
	cluster := &testCluster{}
	ls := &localStore{kv: map[string]any{}}
	handled := false

	contract := &testContract{
		id:     "cid",
		input:  &bridgeIn{},
		output: &bridgeOut{},
		handlers: []HandlerFunc{
			func(ctx *Context) {
				in := ctx.In().GetMsg().(*bridgeIn) //nolint:forcetypeassert
				if in.Name != "value" {
					t.Fatalf("unexpected input: %s", in.Name)
				}
				handled = true
				ctx.Out().SetMsg(&bridgeOut{OK: true}).Send()
			},
		},
	}

	sb := &southBridge{
		ctxPool:      ctxPool{ls: ls},
		id:           "target",
		wg:           &sync.WaitGroup{},
		eh:           func(_ *Context, err error) { t.Fatalf("unexpected error: %v", err) },
		c:            map[string]Contract{"cid": contract},
		cb:           cluster,
		inProgress:   map[string]*clusterConn{},
		msgFactories: map[string]MessageFactoryFunc{},
	}
	sb.registerContract(&bridgeIn{}, &bridgeOut{})

	origCtx := newContext(ls)
	origCtx.conn = newTestConn()
	origCtx.in = newEnvelope(origCtx, origCtx.conn, false)
	origCtx.in.SetID("env").SetMsg(&bridgeIn{Name: "value"})
	origCtx.setServiceName("svc").setContractID("cid").setRoute("route")
	origCtx.sb = sb

	carrier := newEnvelopeCarrier(incomingCarrier, "session", "origin", "target").FillWithContext(origCtx)
	sb.onIncomingMessage(carrier)

	if !handled {
		t.Fatal("expected handler to run")
	}
	if len(cluster.published) != 2 {
		t.Fatalf("unexpected publish count: %d", len(cluster.published))
	}

	kinds := map[carrierKind]bool{}
	for _, item := range cluster.published {
		dec := &envelopeCarrier{}
		err := defaultMessageCodec.Decode(dec, bytes.NewReader(item.data))
		if err != nil {
			t.Fatalf("failed to decode carrier: %v", err)
		}
		kinds[dec.Kind] = true
	}
	if !kinds[outgoingCarrier] || !kinds[eofCarrier] {
		t.Fatalf("unexpected carrier kinds: %#v", kinds)
	}
}

func TestSouthBridgeGenCallback(t *testing.T) {
	ls := &localStore{kv: map[string]any{}}
	ctx := newContext(ls)
	conn := newTestConn()
	ctx.conn = conn
	ctx.in = newEnvelope(ctx, conn, false)

	sb := &southBridge{}
	cb := sb.genCallback(ctx)
	carrier := &envelopeCarrier{Data: &carrierData{
		EnvelopeID: "id",
		StatusCode: 202,
		Hdr:        map[string]string{"k": "v"},
		Msg:        []byte("raw"),
		ConnHdr:    map[string]string{"ck": "cv"},
	}}
	cb(carrier)

	if got := conn.Get("ck"); got != "cv" {
		t.Fatalf("unexpected conn header: %s", got)
	}
	if len(conn.out) != 1 {
		t.Fatalf("unexpected envelope count: %d", len(conn.out))
	}
	if conn.out[0].GetHdr("k") != "v" {
		t.Fatalf("unexpected envelope header: %s", conn.out[0].GetHdr("k"))
	}
	if string(conn.out[0].GetMsg().(RawMessage)) != "raw" { //nolint:forcetypeassert
		t.Fatalf("unexpected envelope message: %v", conn.out[0].GetMsg())
	}
}

func TestSouthBridgeWrapWithCoordinator(t *testing.T) {
	sb := &southBridge{id: "node"}
	contract := &testContract{
		id: "c1",
		edgeSel: func(*LimitedContext) (string, error) {
			return "peer", nil
		},
		handlers: []HandlerFunc{
			func(*Context) {},
		},
	}

	wrapped := sb.wrapWithCoordinator(contract)
	if len(wrapped.Handlers()) != 2 {
		t.Fatalf("unexpected handlers count: %d", len(wrapped.Handlers()))
	}

	plain := (*southBridge)(nil).wrapWithCoordinator(contract)
	if plain != contract {
		t.Fatal("expected nil bridge to return contract")
	}
}

func TestClusterConnKV(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &clusterConn{
		ctx: ctx,
		cf:  cancel,
		kv:  map[string]string{},
	}

	if c.ConnID() != 0 {
		t.Fatalf("unexpected ConnID: %d", c.ConnID())
	}
	c.clientIP = "1.2.3.4"
	if c.ClientIP() != "1.2.3.4" {
		t.Fatalf("unexpected ClientIP: %s", c.ClientIP())
	}
	c.stream = true
	if !c.Stream() {
		t.Fatal("expected Stream to return true")
	}

	c.Set("k", "v")
	if c.Get("k") != "v" {
		t.Fatalf("unexpected value: %s", c.Get("k"))
	}
	keys := c.Keys()
	if len(keys) != 1 || keys[0] != "k" {
		t.Fatalf("unexpected keys: %#v", keys)
	}
	walked := false
	c.Walk(func(key string, val string) bool {
		walked = true

		return true
	})
	if !walked {
		t.Fatal("expected Walk to be called")
	}

	c.Cancel()
	select {
	case <-c.Done():
	default:
		t.Fatal("expected Done to be closed")
	}
	if c.Err() == nil {
		t.Fatal("expected context error")
	}
}
