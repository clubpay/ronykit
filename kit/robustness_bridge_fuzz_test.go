package kit

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func FuzzSouthBridgeOnIncomingMessageRawNoPanic(f *testing.F) {
	f.Add("sid", "origin", "target", "svc", "cid", "route", []byte("payload"))
	f.Add("", "", "", "", "", "", []byte{})

	f.Fuzz(func(
		t *testing.T,
		sessionID, originID, targetID, serviceName, contractID, route string,
		payload []byte,
	) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("onIncomingMessage panicked: %v", r)
			}
		}()

		ls := &localStore{kv: map[string]any{}}
		sb := &southBridge{
			ctxPool:      ctxPool{ls: ls},
			id:           "node-A",
			wg:           &sync.WaitGroup{},
			eh:           func(*Context, error) {},
			cb:           &testCluster{},
			inProgress:   map[string]*clusterConn{},
			msgFactories: map[string]MessageFactoryFunc{},
			c:            map[string]Contract{},
		}
		sb.msgFactories[reflect.TypeOf(RawMessage{}).String()] = CreateMessageFactory(RawMessage{})

		carrier := &envelopeCarrier{
			SessionID: sessionID,
			Kind:      incomingCarrier,
			OriginID:  originID,
			TargetID:  targetID,
			Data: &carrierData{
				EnvelopeID:  "env",
				MsgType:     reflect.TypeOf(RawMessage{}).String(),
				Msg:         payload,
				ServiceName: serviceName,
				ContractID:  contractID,
				Route:       route,
				ConnHdr:     map[string]string{"k": "v"},
				Hdr:         map[string]string{"h": "1"},
			},
		}

		sb.onIncomingMessage(carrier)
	})
}

func FuzzForwarderHandlerNoPanic(f *testing.F) {
	f.Add("", int64(1), []byte("x"), false)
	f.Add("node-A", int64(1), []byte("x"), false)
	f.Add("node-B", int64(1), []byte(`{"a":1}`), false)
	f.Add("node-B", int64(1), []byte(`{"a":1}`), true)

	f.Fuzz(func(t *testing.T, target string, timeoutNanos int64, payload []byte, publishErr bool) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("genForwarderHandler panicked: %v", r)
			}
		}()

		if timeoutNanos <= 0 {
			timeoutNanos = 1
		}
		if timeoutNanos > int64(5*time.Millisecond) {
			timeoutNanos = int64(5 * time.Millisecond)
		}

		cluster := &testCluster{}
		if publishErr {
			cluster.publishErr = errFuzzPublish
		}

		ls := &localStore{kv: map[string]any{}}
		sb := &southBridge{
			ctxPool:      ctxPool{ls: ls},
			id:           "node-A",
			wg:           &sync.WaitGroup{},
			eh:           func(*Context, error) {},
			cb:           cluster,
			inProgress:   map[string]*clusterConn{},
			msgFactories: map[string]MessageFactoryFunc{},
		}

		h := sb.genForwarderHandler(func(*LimitedContext) (string, error) {
			return target, nil
		})

		conn := newTestConn()
		ctx := newContext(ls)
		ctx.conn = conn
		ctx.in = newEnvelope(ctx, conn, false)
		ctx.in.SetMsg(RawMessage(payload))
		ctx.setServiceName("svc").setContractID("cid").setRoute("route")
		ctx.sb = sb
		ctx.rxt = time.Duration(timeoutNanos)

		h(ctx)
	})
}

var errFuzzPublish = &fuzzPublishError{}

type fuzzPublishError struct{}

func (f *fuzzPublishError) Error() string { return "fuzz publish error" }
