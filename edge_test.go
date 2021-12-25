package ronykit_test

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/ronaksoft/ronykit"
	. "github.com/smartystreets/goconvey/convey"
)

type testConn struct {
	kv map[string]interface{}
	id uint64
	ip string
}

func newTestConn() *testConn {
	return &testConn{
		kv: map[string]interface{}{},
		id: gofakeit.Uint64(),
		ip: gofakeit.IPv4Address(),
	}
}

func (t testConn) ConnID() uint64 {
	return t.id
}

func (t testConn) ClientIP() string {
	return t.ip

}

func (t testConn) Write(streamID int64, data []byte) error {
	return nil
}

func (t testConn) Stream() bool {
	return false
}

func (t testConn) Walk(f func(key string, val interface{}) bool) {
	for k, v := range t.kv {
		if !f(k, v) {
			return
		}
	}

	return
}

func (t testConn) Get(key string) interface{} {
	return t.kv[key]
}

func (t testConn) Set(key string, val interface{}) {
	t.kv[key] = val
}

type testGateway struct {
	d ronykit.GatewayDelegate
}

func (t testGateway) Start() {}

func (t testGateway) Shutdown() {}

func (t *testGateway) Subscribe(d ronykit.GatewayDelegate) {
	t.d = d
}

func (t *testGateway) Send(msg []byte) error {
	c := newTestConn()
	return t.d.OnMessage(c, 0, msg)
}

type testMessage []byte

func (t testMessage) Unmarshal(bytes []byte) error {
	return nil
}

func (t testMessage) Marshal() ([]byte, error) {
	return t, nil
}

type testDispatcher struct {
}

func (t testDispatcher) Dispatch(conn ronykit.Conn, streamID int64, in []byte) ronykit.DispatchFunc {
	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		execFunc(
			testMessage(in),
			func(m ronykit.Message) error {
				b, err := m.Marshal()
				if err != nil {
					return err
				}
				return conn.Write(streamID, b)
			},
			func(ctx *ronykit.Context) ronykit.Handler {
				return func(ctx *ronykit.Context) ronykit.Handler {
					m := ctx.Receive()
					_ = ctx.Send(m)
					return nil
				}
			},
		)

		return nil
	}
}

type testBundle struct {
	gw *testGateway
	d  *testDispatcher
}

func (t testBundle) Gateway() ronykit.Gateway {
	return t.gw
}

func (t testBundle) Dispatcher() ronykit.Dispatcher {
	return t.d
}

func TestServer(t *testing.T) {
	Convey("Test Server", t, func(c C) {
		b := &testBundle{
			gw: &testGateway{},
			d:  &testDispatcher{},
		}
		s := ronykit.NewServer(
			ronykit.RegisterBundle(b),
		)
		s.Start()
		err := b.gw.Send([]byte("123"))
		c.So(err, ShouldBeNil)
		s.Shutdown()
	})
}

func BenchmarkServer(b *testing.B) {
	bundle := &testBundle{
		gw: &testGateway{},
		d:  &testDispatcher{},
	}
	s := ronykit.NewServer(
		ronykit.RegisterBundle(bundle),
	)
	s.Start()
	defer s.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(
		func(pb *testing.PB) {
			for pb.Next() {
				_ = bundle.gw.Send([]byte("123"))
			}
		},
	)
}
