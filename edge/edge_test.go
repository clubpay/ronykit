package edge_test

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/edge"
	. "github.com/smartystreets/goconvey/convey"
)

type testConn struct {
	id       uint64
	clientIP string
	bag      map[string]interface{}
	cb       func(streamID int64, data []byte)
	stream   bool
}

func newConn(stream bool, cb func(streamID int64, data []byte)) *testConn {
	return &testConn{
		id:       gofakeit.Uint64(),
		clientIP: gofakeit.IPv4Address(),
		bag:      map[string]interface{}{},
		cb:       cb,
		stream:   stream,
	}
}

func (t testConn) ConnID() uint64 {
	return t.id
}

func (t testConn) ClientIP() string {
	return t.clientIP
}

func (t testConn) Write(streamID int64, data []byte) error {
	if t.cb != nil {
		t.cb(streamID, data)
	}

	return nil
}

func (t testConn) Stream() bool {
	return t.stream
}

func (t testConn) Walk(f func(key string, val interface{}) bool) {
	for k, v := range t.bag {
		if !f(k, v) {
			return
		}
	}
}

func (t testConn) Get(key string) interface{} {
	return t.bag[key]
}

func (t testConn) Set(key string, val interface{}) {
	t.bag[key] = val
}

type testGateway struct {
	sync.Mutex
	conns map[uint64]*testConn
	d     ronykit.GatewayDelegate
}

func newGateway() *testGateway {
	return &testGateway{
		conns: map[uint64]*testConn{},
	}
}

func (t *testGateway) Start() {
	// Nothing to do
}

func (t *testGateway) Shutdown() {
	// Nothing to do
}

func (t *testGateway) Subscribe(d ronykit.GatewayDelegate) {
	t.d = d
}

func (t *testGateway) OpenConn(stream bool, cb func(streamID int64, data []byte)) uint64 {
	c := newConn(stream, cb)

	t.Lock()
	t.conns[c.ConnID()] = c
	t.Unlock()

	t.d.OnOpen(c)

	return c.ConnID()
}

func (t *testGateway) Send(connID uint64, streamID int64, data []byte) error {
	t.Lock()
	c := t.conns[connID]
	t.Unlock()

	return t.d.OnMessage(c, streamID, data)
}

func (t *testGateway) CloseConn(connID uint64) {
	t.Lock()
	_, ok := t.conns[connID] // nolint:ifshort
	delete(t.conns, connID)
	t.Unlock()

	if ok {
		t.d.OnClose(connID)
	}
}

type testEnvelope struct {
	sync.Mutex
	Subject string
	Body    []byte
	HDR     map[string]string
}

func newEnvelope() *testEnvelope {
	return &testEnvelope{
		HDR: map[string]string{},
	}
}

func (t *testEnvelope) Get(key string) (string, bool) {
	t.Lock()
	x, ok := t.HDR[key]
	t.Unlock()

	return x, ok
}

func (t *testEnvelope) Set(key, val string) {
	t.Lock()
	t.HDR[key] = val
	t.Unlock()
}

func (t *testEnvelope) Header() ronykit.EnvelopeHeader {
	return t
}

func (t *testEnvelope) SetSubject(s string) {
	t.Subject = s
}

func (t *testEnvelope) GetSubject() string {
	return t.Subject
}

func (t *testEnvelope) SetBody(bytes []byte) {
	t.Body = bytes
}

func (t *testEnvelope) GetBody() []byte {
	return t.Body
}

type testEnvelopeContainer struct {
	Envelopes []ronykit.Envelope
}

type testDispatcher struct{}

func (t testDispatcher) Serialize(conn ronykit.Conn, streamID int64, envelopes ...ronykit.Envelope) error {
	switch len(envelopes) {
	case 0:
	case 1:
		b, err := json.Marshal(envelopes[0])
		if err != nil {
			return err
		}

		return conn.Write(streamID, b)
	default:
		ec := &testEnvelopeContainer{}
		ec.Envelopes = append(ec.Envelopes, envelopes...)

		b, err := json.Marshal(envelopes[0])
		if err != nil {
			return err
		}

		return conn.Write(streamID, b)
	}

	return nil
}

func (t testDispatcher) Deserialize(conn ronykit.Conn, data []byte, f func(envelope ronykit.Envelope) error) error {
	e := newEnvelope()
	if err := json.Unmarshal(data, e); err != nil {
		return err
	}

	return f(e)
}

func (t testDispatcher) OnOpen(conn ronykit.Conn) {
	// Nothing to do
}

type testRouter struct{}

func (t testRouter) Route(envelope ronykit.Envelope) ([]edge.Handler, error) {
	var handlers []edge.Handler
	switch envelope.GetSubject() {
	case "echo":
		handlers = append(handlers, echoHandler)
	default:
		return nil, fmt.Errorf("no handler")
	}

	return handlers, nil
}

func echoHandler(ctx *edge.RequestCtx, e ronykit.Envelope) edge.Handler {
	ctx.Push(e)

	return nil
}

func TestServer_RegisterGateway(t *testing.T) {
	Convey("Edge Server Tests", t, func(c C) {
		Convey("RegisterGateway", func(c C) {
			gw := newGateway()
			es := edge.New(testRouter{}, nil)
			es.RegisterGateway(gw, testDispatcher{})
			es.Start()

			reqStreamID := gofakeit.Int64()
			req := newEnvelope()
			req.Subject = "echo"
			req.Body = []byte("This is some random data")
			reqBytes, err := json.Marshal(req)
			c.So(err, ShouldBeNil)

			res := newEnvelope()

			wg := sync.WaitGroup{}
			wg.Add(1)
			connID := gw.OpenConn(false,
				func(streamID int64, data []byte) {
					err := json.Unmarshal(data, res)
					c.So(err, ShouldBeNil)
					c.So(streamID, ShouldEqual, reqStreamID)
					wg.Done()
				},
			)
			err = gw.Send(connID, reqStreamID, reqBytes)
			c.So(err, ShouldBeNil)

			wg.Wait()
			c.So(res.Subject, ShouldEqual, req.Subject)
			c.So(res.Body, ShouldResemble, req.Body)

			es.Shutdown()
		})
	})
}

func BenchmarkServer(b *testing.B) {
	gw := newGateway()
	es := edge.New(testRouter{}, nil)
	es.RegisterGateway(gw, testDispatcher{})
	es.Start()

	reqStreamID := gofakeit.Int64()
	req := newEnvelope()
	req.Subject = "echo"
	req.Body = []byte("This is some random data")
	reqBytes, err := json.Marshal(req)
	if err != nil {
		b.Fatal(err)
	}

	wg := sync.WaitGroup{}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(
		func(pb *testing.PB) {
			for pb.Next() {
				res := newEnvelope()

				wg.Add(1)
				connID := gw.OpenConn(false,
					func(streamID int64, data []byte) {
						err := json.Unmarshal(data, res)
						if err != nil {
							b.Fatal(err)
						}

						wg.Done()
					},
				)
				err = gw.Send(connID, reqStreamID, reqBytes)
				if err != nil {
					b.Fatal(err)
				}
				gw.CloseConn(connID)
			}
		},
	)

	wg.Wait()

	es.Shutdown()
}
