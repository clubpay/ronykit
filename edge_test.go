package ronykit_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/utils"
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
		id: utils.RandomUint64(0),
		ip: "127.0.0.1",
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

func (t *testGateway) Send(msg []byte) {
	c := newTestConn()

	t.d.OnMessage(c, msg)
}

type testMessage []byte

func (t testMessage) Marshal() ([]byte, error) {
	return t, nil
}

type testDispatcher struct{}

func (t testDispatcher) Dispatch(ctx *ronykit.Context, in []byte, execFunc ronykit.ExecuteFunc) error {
	ctx.In().SetMsg(testMessage(in))
	execFunc(
		ctx,
		func(conn ronykit.Conn, e *ronykit.Envelope) error {
			b, err := json.Marshal(e.GetMsg())
			if err != nil {
				return err
			}

			_, err = conn.Write(b)

			return err
		},
		func(ctx *ronykit.Context) {
			m := ctx.In().GetMsg()

			ctx.Out().
				SetMsg(m).
				Send()

			return
		},
	)

	return nil
}

type testBundle struct {
	gw *testGateway
	d  *testDispatcher
}

func (t testBundle) Start(_ context.Context) error {
	t.gw.Start()

	return nil
}

func (t testBundle) Shutdown(_ context.Context) error {
	t.gw.Shutdown()

	return nil
}

func (t testBundle) Subscribe(d ronykit.GatewayDelegate) {
	t.gw.Subscribe(d)
}

func (t testBundle) Dispatch(ctx *ronykit.Context, in []byte, execFunc ronykit.ExecuteFunc) error {
	return t.d.Dispatch(ctx, in, execFunc)
}

func (t testBundle) Register(srv ronykit.Service) {}

func TestServer(t *testing.T) {
	Convey("Test EdgeServer", t, func(c C) {
		b := &testBundle{
			gw: &testGateway{},
			d:  &testDispatcher{},
		}
		s := ronykit.NewServer(
			ronykit.RegisterGateway(b),
		)
		s.Start(nil)
		b.gw.Send([]byte("123"))
		s.Shutdown(nil)
	})
}

func BenchmarkServer(b *testing.B) {
	bundle := &testBundle{
		gw: &testGateway{},
		d:  &testDispatcher{},
	}
	s := ronykit.NewServer(
		ronykit.RegisterGateway(bundle),
	).Start(nil)
	defer s.Shutdown(nil)

	req := []byte(utils.RandomID(24))
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(
		func(pb *testing.PB) {
			for pb.Next() {
				bundle.gw.Send(req)
			}
		},
	)
}
