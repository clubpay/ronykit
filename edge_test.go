package ronykit_test

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/ronaksoft/ronykit"
	. "github.com/smartystreets/goconvey/convey"
)

type testConn struct {
	kv map[string]string
	id uint64
	ip string
}

func newTestConn() *testConn {
	return &testConn{
		kv: map[string]string{},
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

func (t testConn) Write(data []byte) (int, error) {
	return 0, nil
}

func (t testConn) Stream() bool {
	return false
}

func (t testConn) Walk(f func(key string, val string) bool) {
	for k, v := range t.kv {
		if !f(k, v) {
			return
		}
	}

	return
}

func (t testConn) Get(key string) string {
	return t.kv[key]
}

func (t testConn) Set(key string, val string) {
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
	return t.d.OnMessage(c, msg)
}

type testMessage []byte

func (t testMessage) Unmarshal(bytes []byte) error {
	return nil
}

func (t testMessage) Marshal() ([]byte, error) {
	return t, nil
}

type testDispatcher struct{}

func (t testDispatcher) Dispatch(conn ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		execFunc(
			testMessage(in),
			func(m ronykit.Message) {
				b, err := m.Marshal()
				if err != nil {
					ctx.Error(err)

					return
				}

				_, err = conn.Write(b)
				ctx.Error(err)
			},
			func(ctx *ronykit.Context) ronykit.Handler {
				return func(ctx *ronykit.Context) ronykit.Handler {
					m := ctx.Receive()
					ctx.Send(m)

					return nil
				}
			},
		)

		return nil
	}, nil
}

type testBundle struct {
	gw *testGateway
	d  *testDispatcher
}

func (t testBundle) Start() {
	t.gw.Start()
}

func (t testBundle) Shutdown() {
	t.gw.Shutdown()
}

func (t testBundle) Subscribe(d ronykit.GatewayDelegate) {
	t.gw.Subscribe(d)
}

func (t testBundle) Dispatch(conn ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	return t.d.Dispatch(conn, in)
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
	).Start()
	defer s.Shutdown()

	req := []byte(gofakeit.BitcoinAddress())
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(
		func(pb *testing.PB) {
			for pb.Next() {
				_ = bundle.gw.Send(req)
			}
		},
	)
}
