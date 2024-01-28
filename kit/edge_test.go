package kit_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRonykit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ronykit Suite")
}

type testSelector struct{}

func (t testSelector) Query(_ string) any {
	return nil
}

func (t testSelector) GetEncoding() kit.Encoding {
	return kit.JSON
}

type testGateway struct {
	d kit.GatewayDelegate
}

var _ kit.Gateway = (*testGateway)(nil)

func (t *testGateway) Send(c *testConn, msg []byte) {
	t.d.OnMessage(
		c,
		func(conn kit.Conn, e *kit.Envelope) error {
			b, err := kit.MarshalMessage(e.GetMsg())
			if err != nil {
				return err
			}

			_, err = conn.Write(b)

			return err
		},
		msg,
	)
}

func (t *testGateway) Start(_ context.Context, _ kit.GatewayStartConfig) error {
	return nil
}

func (t *testGateway) Shutdown(_ context.Context) error {
	return nil
}

func (t *testGateway) Subscribe(d kit.GatewayDelegate) {
	t.d = d
}

func (t *testGateway) Dispatch(ctx *kit.Context, in []byte) (kit.ExecuteArg, error) {
	ctx.In().SetMsg(kit.RawMessage(in))

	return kit.ExecuteArg{
		ServiceName: "testService",
		ContractID:  "testService.1",
		Route:       "someRoute",
	}, nil
}

func (t *testGateway) Register(
	_, _ string, _ kit.Encoding, _ kit.RouteSelector, _ kit.Message,
) {
}

type testCluster struct {
	mtx       sync.Mutex
	delegates map[string]kit.ClusterDelegate
	kv        map[string]string
	m         chan struct {
		id   string
		data []byte
	}
}

var (
	_ kit.Cluster      = (*testCluster)(nil)
	_ kit.ClusterStore = (*testCluster)(nil)
)

func newTestCluster() *testCluster {
	t := &testCluster{
		delegates: map[string]kit.ClusterDelegate{},
		m: make(chan struct {
			id   string
			data []byte
		}, 10),
	}

	go func() {
		for x := range t.m {
			t.mtx.Lock()
			d, ok := t.delegates[x.id]
			t.mtx.Unlock()
			if ok {
				_ = d.OnMessage(x.data)
			}
		}
	}()

	return t
}

func (t *testCluster) Start(ctx context.Context) error {
	return nil
}

func (t *testCluster) Shutdown(ctx context.Context) error {
	return nil
}

func (t *testCluster) Subscribe(id string, d kit.ClusterDelegate) {
	t.mtx.Lock()
	if t.delegates == nil {
		t.delegates = map[string]kit.ClusterDelegate{}
	}
	t.delegates[id] = d
	t.mtx.Unlock()
}

func (t *testCluster) Subscribers() ([]string, error) {
	var members []string
	t.mtx.Lock()
	for m := range t.delegates {
		members = append(members, m)
	}
	t.mtx.Unlock()

	return members, nil
}

func (t *testCluster) Publish(id string, data []byte) error {
	t.m <- struct {
		id   string
		data []byte
	}{id: id, data: data}

	return nil
}

func (t *testCluster) Store() kit.ClusterStore {
	return t
}

func (t *testCluster) Set(key, value string, _ time.Duration) error {
	if t.kv == nil {
		t.kv = map[string]string{}
	}

	t.kv[key] = value

	return nil
}

func (t *testCluster) Delete(key string) error {
	if t.kv == nil {
		t.kv = map[string]string{}
	}

	delete(t.kv, key)

	return nil
}

func (t *testCluster) Get(key string) (string, error) {
	return t.kv[key], nil
}

func (t *testCluster) Scan(prefix string, cb func(key string) bool) error {
	for k := range t.kv {
		if strings.HasPrefix(k, prefix) {
			if !cb(k) {
				return nil
			}
		}
	}

	return nil
}

func (t *testCluster) ScanWithValue(prefix string, cb func(string, string) bool) error {
	for k, v := range t.kv {
		if strings.HasPrefix(k, prefix) {
			if !cb(k, v) {
				return nil
			}
		}
	}

	return nil
}

type testConn struct {
	id       uint64
	clientIP string
	stream   bool
	kv       map[string]string
	buf      *bytes.Buffer
}

var _ kit.Conn = (*testConn)(nil)

func newTestConn(id uint64, clientIP string, stream bool) *testConn {
	return &testConn{
		id:       id,
		clientIP: clientIP,
		stream:   stream,
		kv:       map[string]string{},
		buf:      &bytes.Buffer{},
	}
}

func (t testConn) ConnID() uint64 {
	return t.id
}

func (t testConn) ClientIP() string {
	return t.clientIP
}

func (t testConn) Write(data []byte) (int, error) {
	return t.buf.Write(data)
}

func (t testConn) Read() ([]byte, error) {
	return io.ReadAll(t.buf)
}

func (t testConn) ReadString() (string, error) {
	d, err := t.Read()
	if err != nil {
		return "", err
	}
	return string(d), nil
}

func (t testConn) Stream() bool {
	return t.stream
}

func (t testConn) Walk(f func(key string, val string) bool) {
	for k, v := range t.kv {
		if !f(k, v) {
			return
		}
	}
}

func (t testConn) Get(key string) string {
	return t.kv[key]
}

func (t *testConn) Set(key string, val string) {
	t.kv[key] = val
}

func (t testConn) Keys() []string {
	keys := make([]string, 0, len(t.kv))
	for k := range t.kv {
		keys = append(keys, k)
	}

	return keys
}

// testRESTSelector implements kit.RESTRouteSelector for testing purposes.
type testRESTSelector struct {
	enc    kit.Encoding
	method string
	path   string
}

func (t testRESTSelector) Query(q string) any {
	return nil
}

func (t testRESTSelector) GetEncoding() kit.Encoding {
	return t.enc
}

func (t testRESTSelector) GetMethod() string {
	return t.method
}

func (t testRESTSelector) GetPath() string {
	return t.path
}

// testRPCSelector implements kit.RPCSelector for testing purposes.
type testRPCSelector struct {
	enc       kit.Encoding
	predicate string
}

func (t testRPCSelector) Query(_ string) any {
	return nil
}

func (t testRPCSelector) GetEncoding() kit.Encoding {
	return t.enc
}

func (t testRPCSelector) GetPredicate() string {
	return t.predicate
}

var _ = Describe("EdgeServer/Simple", func() {
	var (
		b    *testGateway
		edge *kit.EdgeServer
	)
	BeforeEach(func() {
		b = &testGateway{}
		var serviceDesc desc.ServiceDescFunc = func() *desc.Service {
			return desc.NewService("testService").
				AddContract(
					desc.NewContract().
						SetInput(&kit.RawMessage{}).
						SetOutput(&kit.RawMessage{}).
						AddSelector(testSelector{}).
						AddHandler(
							func(ctx *kit.Context) {
								ctx.Out().
									SetMsg(ctx.In().GetMsg()).
									Send()

								return
							},
						),
				)
		}
		edge = kit.NewServer(
			kit.WithGateway(b),
			kit.WithServiceDesc(serviceDesc.Desc()),
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
		Entry("a raw string", kit.RawMessage("Hello this is a simple message")),
		Entry("a ToJSON string", kit.RawMessage(`{"cmd": "something", "key1": 123, "key2": "val2"}`)),
	)
})

var _ = Describe("EdgeServer/GlobalHandlers", func() {
	var (
		b    *testGateway
		edge *kit.EdgeServer
	)
	BeforeEach(func() {
		b = &testGateway{}
		var serviceDesc desc.ServiceDescFunc = func() *desc.Service {
			return desc.NewService("testService").
				AddContract(
					desc.NewContract().
						SetInput(&kit.RawMessage{}).
						SetOutput(&kit.RawMessage{}).
						AddSelector(testSelector{}).
						AddHandler(
							func(ctx *kit.Context) {
								in := utils.B2S(ctx.In().GetMsg().(kit.RawMessage)) //nolint:forcetypeassert
								out := fmt.Sprintf("%s-%s-%s",
									ctx.GetString("PRE_KEY", ""),
									in,
									ctx.GetString("POST_KEY", ""),
								)
								ctx.Out().
									SetMsg(kit.RawMessage(out)).
									Send()

								return
							},
						),
				)
		}
		edge = kit.NewServer(
			kit.WithGateway(b),
			kit.WithGlobalHandlers(
				func(ctx *kit.Context) {
					ctx.Set("PRE_KEY", "PRE_VALUE")
				},
			),
			kit.WithServiceDesc(serviceDesc.Desc()),
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
		Entry("a raw string", kit.RawMessage("Hello this is a simple message")),
		Entry("a ToJSON string", kit.RawMessage(`{"cmd": "something", "key1": 123, "key2": "val2"}`)),
	)
})

var _ = Describe("EdgeServer/Cluster", func() {
	var (
		b1    *testGateway
		b2    *testGateway
		c     *testCluster
		edge1 *kit.EdgeServer
		edge2 *kit.EdgeServer
	)
	BeforeEach(func() {
		b1 = &testGateway{}
		b2 = &testGateway{}
		c = newTestCluster()

		var serviceDesc = func(id string) desc.ServiceDescFunc {
			return func() *desc.Service {
				return desc.NewService("testService").
					AddContract(
						desc.NewContract().
							SetInput(kit.RawMessage{}).
							SetOutput(kit.RawMessage{}).
							AddSelector(testSelector{}).
							SetCoordinator(
								func(ctx *kit.LimitedContext) (string, error) {
									members, err := ctx.ClusterMembers()
									if err != nil {
										return "", err
									}

									for _, m := range members {
										if m != ctx.ClusterID() {
											return m, nil
										}
									}

									return ctx.ClusterID(), nil
								},
							).
							AddHandler(
								func(ctx *kit.Context) {
									ctx.Out().
										SetMsg(kit.RawMessage(id)).
										Send()

									return
								},
							),
					)
			}
		}
		edge1 = kit.NewServer(
			kit.WithGateway(b1),
			kit.WithCluster(c),
			kit.WithServiceDesc(serviceDesc("edge1").Desc()),
		)
		edge1.Start(nil)
		edge2 = kit.NewServer(
			kit.WithGateway(b2),
			kit.WithCluster(c),
			kit.WithServiceDesc(serviceDesc("edge2").Desc()),
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
			c.Set("X-Hdr1", "edge1")
			b1.Send(c, msg)
			Expect(c.ReadString()).To(BeEquivalentTo("edge2"))
			Expect(c.Get("X-Hdr1")).To(BeEquivalentTo("edge1"))
		},
		Entry("a raw string", kit.RawMessage("Hello this is a simple message")),
		Entry("a ToJSON string", kit.RawMessage(`{"cmd": "something", "key1": 123, "key2": "val2"}`)),
	)
})

func BenchmarkServer(b *testing.B) {
	bundle := &testGateway{}
	s := kit.NewServer(
		kit.WithGateway(bundle),
		kit.WithService(
			desc.NewService("testService").
				AddContract(
					desc.NewContract().
						AddSelector(testSelector{}).
						AddHandler(
							func(ctx *kit.Context) {
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
