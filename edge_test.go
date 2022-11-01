package ronykit_test

import (
	"context"
	"fmt"
	"sync"
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
	t.d.OnMessage(
		c,
		func(conn ronykit.Conn, e *ronykit.Envelope) error {
			b, err := ronykit.MarshalMessage(e.GetMsg())
			if err != nil {
				return err
			}

			_, err = conn.Write(b)

			return err
		},
		msg,
	)
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
		ServiceName: "testService",
		ContractID:  "testService.1",
		Route:       "someRoute",
	}, nil
}

func (t testGateway) Register(
	serviceName, contractID string, enc ronykit.Encoding, sel ronykit.RouteSelector, input ronykit.Message,
) {
}

type testCluster struct {
	sync.Mutex
	delegates map[string]ronykit.ClusterDelegate
}

func (t *testCluster) Start(ctx context.Context) error {
	return nil
}

func (t *testCluster) Shutdown(ctx context.Context) error {
	return nil
}

func (t *testCluster) Subscribe(id string, d ronykit.ClusterDelegate) {
	fmt.Println("Subscribe: ", id)

	t.Lock()
	if t.delegates == nil {
		t.delegates = map[string]ronykit.ClusterDelegate{}
	}
	t.delegates[id] = d
	t.Unlock()
}

func (t *testCluster) Publish(id string, data []byte) error {
	fmt.Println("Publish: ", id, string(data))

	t.Lock()
	d, ok := t.delegates[id]
	t.Unlock()
	if ok {
		d.OnMessage(data)
	}

	return nil
}

var _ = Describe("EdgeServer/Simple", func() {
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
		Entry("a ToJSON string", ronykit.RawMessage(`{"cmd": "something", "key1": 123, "key2": "val2"}`)),
	)
})

var _ = Describe("EdgeServer/GlobalHandlers", func() {
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
								in := utils.B2S(ctx.In().GetMsg().(ronykit.RawMessage))
								out := fmt.Sprintf("%s-%s-%s",
									ctx.GetString("PRE_KEY", ""),
									in,
									ctx.GetString("POST_KEY", ""),
								)
								ctx.Out().
									SetMsg(ronykit.RawMessage(out)).
									Send()

								return
							},
						),
				)
		}
		edge = ronykit.NewServer(
			ronykit.RegisterBundle(b),
			ronykit.WithGlobalHandlers(
				func(ctx *ronykit.Context) {
					ctx.Set("PRE_KEY", "PRE_VALUE")
				},
			),
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
			Expect(c.Read()).To(BeEquivalentTo(fmt.Sprintf("PRE_VALUE-%s-", string(msg))))
		},
		Entry("a raw string", ronykit.RawMessage("Hello this is a simple message")),
		Entry("a ToJSON string", ronykit.RawMessage(`{"cmd": "something", "key1": 123, "key2": "val2"}`)),
	)
})

var _ = Describe("EdgeServer/Cluster", func() {
	var (
		b1    *testGateway
		b2    *testGateway
		c     *testCluster
		edge1 *ronykit.EdgeServer
		edge2 *ronykit.EdgeServer
	)
	BeforeEach(func() {
		b1 = &testGateway{}
		b2 = &testGateway{}
		c = &testCluster{}
		var serviceDesc = func(id string) desc.ServiceDescFunc {
			return func() *desc.Service {
				return desc.NewService("testService").
					AddContract(
						desc.NewContract().
							SetInput(&ronykit.RawMessage{}).
							SetOutput(&ronykit.RawMessage{}).
							AddSelector(testSelector{}).
							SetCoordinator(
								func(ctx *ronykit.LimitedContext) (string, error) {
									return "edge2", nil
								},
							).
							AddHandler(
								func(ctx *ronykit.Context) {
									ctx.Out().
										SetMsg(ronykit.RawMessage(id)).
										Send()

									return
								},
							),
					)
			}
		}
		edge1 = ronykit.NewServer(
			ronykit.RegisterBundle(b1),
			ronykit.RegisterCluster("edge1", c),
			desc.Register(serviceDesc("edge1")),
		)
		edge1.Start(nil)
		edge2 = ronykit.NewServer(
			ronykit.RegisterBundle(b2),
			ronykit.RegisterCluster("edge2", c),
			desc.Register(serviceDesc("edge2")),
		)
		edge2.Start(nil)
	})
	AfterEach(func() {
		edge1.Shutdown(nil)
		edge2.Shutdown(nil)
	})

	DescribeTable("should echo back the message",
		func(msg []byte) {
			c := newTestConn(utils.RandomUint64(0), "", false)
			b1.Send(c, msg)
			Expect(c.Read()).To(BeEquivalentTo(msg))
		},
		Entry("a raw string", ronykit.RawMessage("edge2")),
		Entry("a ToJSON string", ronykit.RawMessage(`edge2`)),
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
