package ronykit_test

import (
	"context"
	"testing"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/desc"
	"github.com/clubpay/ronykit/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testSelector struct{}

func (t testSelector) Query(q string) interface{} {
	return nil
}

func (t testSelector) GetEncoding() ronykit.Encoding {
	return ronykit.JSON
}

type testGateway struct {
	d ronykit.GatewayDelegate
}

var _ ronykit.Gateway = (*testGateway)(nil)

func (t *testGateway) Send(c *testConn, msg []byte) {
	t.d.OnMessage(c, msg)
}

func (t testGateway) Start(_ context.Context) error {
	return nil
}

func (t testGateway) Shutdown(_ context.Context) error {
	return nil
}

func (t *testGateway) Subscribe(d ronykit.GatewayDelegate) {
	t.d = d
}

func (t testGateway) Dispatch(ctx *ronykit.Context, in []byte) (ronykit.ExecuteArg, error) {
	ctx.In().SetMsg(ronykit.RawMessage(in))

	return ronykit.ExecuteArg{
		WriteFunc: func(conn ronykit.Conn, e ronykit.Envelope) error {
			b, err := ronykit.MarshalMessage(e.GetMsg())
			if err != nil {
				return err
			}

			_, err = conn.Write(b)

			return err
		},
		ServiceName: "testService",
		ContractID:  "testService.1",
		Route:       "someRoute",
	}, nil
}

func (t testGateway) Register(
	serviceName, contractID string, enc ronykit.Encoding, sel ronykit.RouteSelector, input ronykit.Message,
) {
}

var _ = Describe("EdgeServer", func() {
	var (
		b    *testGateway
		edge *ronykit.EdgeServer
	)
	BeforeEach(func() {
		b = &testGateway{}
		var serviceDesc desc.ServiceDescFunc = func() *desc.Service {
			return desc.NewService("testService").
				AddContract(
					desc.NewContract().
						SetInput(&ronykit.RawMessage{}).
						SetOutput(&ronykit.RawMessage{}).
						AddSelector(testSelector{}).
						AddHandler(
							func(ctx *ronykit.Context) {
								ctx.Out().
									SetMsg(ctx.In().GetMsg()).
									Send()

								return
							},
						),
				)
		}
		edge = ronykit.NewServer(
			ronykit.RegisterBundle(b),
			desc.Register(serviceDesc),
		)
		edge.Start(nil)
	})
	AfterEach(func() {
		edge.Shutdown(nil)
	})

	DescribeTable("should echo back the message",
		func(msg []byte) {
			c := newTestConn(utils.RandomUint64(0), "", false)
			b.Send(c, msg)
			Expect(c.Read()).To(BeEquivalentTo(msg))
		},
		Entry("a raw string", ronykit.RawMessage("Hello this is a simple message")),
		Entry("a JSON string", ronykit.RawMessage(`{"cmd": "something", "key1": 123, "key2": "val2"}`)),
	)
})

func BenchmarkServer(b *testing.B) {
	bundle := &testGateway{}
	s := ronykit.NewServer(
		ronykit.RegisterBundle(bundle),
		ronykit.RegisterService(
			desc.NewService("testService").
				AddContract(
					desc.NewContract().
						AddSelector(testSelector{}).
						AddHandler(
							func(ctx *ronykit.Context) {
								ctx.Out().
									SetMsg(ctx.In().GetMsg()).
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
				c := newTestConn(utils.RandomUint64(0), "", false)
				bundle.Send(c, req)
			}
		},
	)
}
