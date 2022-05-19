package ronykit_test

import (
	"context"
	"testing"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/desc"
	"github.com/clubpay/ronykit/utils"
	"github.com/goccy/go-json"
)

type testSelector struct{}

func (t testSelector) Query(q string) interface{} {
	return nil
}

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

type testBundle struct {
	gw *testGateway
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

func (t testBundle) Dispatch(ctx *ronykit.Context, in []byte) (ronykit.ExecuteArg, error) {
	ctx.In().SetMsg(testMessage(in))

	return ronykit.ExecuteArg{
		WriteFunc: func(conn ronykit.Conn, e ronykit.Envelope) error {
			b, err := json.Marshal(e.GetMsg())
			if err != nil {
				return err
			}

			_, err = conn.Write(b)

			return err
		},
		ServiceName: "testService",
		ContractID:  "1",
		Route:       "someRoute",
	}, nil
}

func (t testBundle) Register(serviceName, contractID string, sel ronykit.RouteSelector, input ronykit.Message) {
}

func TestServer(t *testing.T) {
	b := &testBundle{
		gw: &testGateway{},
	}
	s := ronykit.NewServer(
		ronykit.RegisterBundle(b),
		ronykit.RegisterService(
			desc.NewService("testService").
				AddContract(
					desc.NewContract().
						AddSelector(testSelector{}).
						AddHandler(
							func(ctx *ronykit.Context) {
								m := ctx.In().GetMsg()

								ctx.Out().
									SetMsg(m).
									Send()

								return
							},
						),
				).
				Generate(),
		),
	)
	s.Start(nil)
	b.gw.Send([]byte("123"))
	s.Shutdown(nil)
}

func BenchmarkServer(b *testing.B) {
	bundle := &testBundle{
		gw: &testGateway{},
	}
	s := ronykit.NewServer(
		ronykit.RegisterBundle(bundle),
		ronykit.RegisterService(
			desc.NewService("testService").
				AddContract(
					desc.NewContract().
						AddSelector(testSelector{}).
						AddHandler(
							func(ctx *ronykit.Context) {
								m := ctx.In().GetMsg()

								ctx.Out().
									SetMsg(m).
									Send()

								return
							},
						),
				).
				Generate(),
		),
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
