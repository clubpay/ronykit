package kit

import (
	"context"
)

type testGateway struct {
	startCalls    int
	shutdownCalls int
	regs          []gatewayReg
	delegate      GatewayDelegate
	dispatchFn    func(ctx *Context, in []byte) (ExecuteArg, error)
	startErr      error
	lastCfg       GatewayStartConfig
}

type gatewayReg struct {
	svc    string
	cid    string
	enc    Encoding
	sel    RouteSelector
	input  Message
	output Message
}

func (g *testGateway) Start(_ context.Context, cfg GatewayStartConfig) error {
	g.startCalls++
	g.lastCfg = cfg

	return g.startErr
}

func (g *testGateway) Shutdown(_ context.Context) error {
	g.shutdownCalls++

	return nil
}

func (g *testGateway) Register(
	svcName, contractID string, enc Encoding, sel RouteSelector, input, output Message,
) {
	g.regs = append(g.regs, gatewayReg{svc: svcName, cid: contractID, enc: enc, sel: sel, input: input, output: output})
}

func (g *testGateway) Subscribe(d GatewayDelegate) {
	g.delegate = d
}

func (g *testGateway) Dispatch(ctx *Context, in []byte) (ExecuteArg, error) {
	if g.dispatchFn != nil {
		return g.dispatchFn(ctx, in)
	}

	return ExecuteArg{}, nil
}

type testCluster struct {
	startCalls    int
	shutdownCalls int
	subscribedID  string
	delegate      ClusterDelegate
	publishErr    error
	published     []clusterPublish
}

type clusterPublish struct {
	id   string
	data []byte
}

func (c *testCluster) Start(_ context.Context) error {
	c.startCalls++

	return nil
}

func (c *testCluster) Shutdown(_ context.Context) error {
	c.shutdownCalls++

	return nil
}

func (c *testCluster) Subscribe(id string, d ClusterDelegate) {
	c.subscribedID = id
	c.delegate = d
}

func (c *testCluster) Publish(id string, data []byte) error {
	c.published = append(c.published, clusterPublish{id: id, data: append([]byte(nil), data...)})

	return c.publishErr
}

func (c *testCluster) Subscribers() ([]string, error) {
	return []string{c.subscribedID}, nil
}

type testRESTSelector struct {
	method   string
	path     string
	encoding Encoding
}

func (r testRESTSelector) Query(string) any      { return nil }
func (r testRESTSelector) GetEncoding() Encoding { return r.encoding }
func (r testRESTSelector) GetMethod() string     { return r.method }
func (r testRESTSelector) GetPath() string       { return r.path }
func (r testRESTSelector) String() string        { return r.method + " " + r.path }

// SetEncoding mirrors the behavior of std selectors in tests.
func (r testRESTSelector) SetEncoding(enc Encoding) testRESTSelector {
	r.encoding = enc

	return r
}

type testRPCSelector struct {
	predicate string
	encoding  Encoding
}

func (r testRPCSelector) Query(string) any      { return nil }
func (r testRPCSelector) GetEncoding() Encoding { return r.encoding }
func (r testRPCSelector) GetPredicate() string  { return r.predicate }
func (r testRPCSelector) String() string        { return r.predicate }

func (r testRPCSelector) SetEncoding(enc Encoding) testRPCSelector {
	r.encoding = enc

	return r
}

type testContract struct {
	id        string
	sel       RouteSelector
	enc       Encoding
	input     Message
	output    Message
	handlers  []HandlerFunc
	modifiers []ModifierFunc
	edgeSel   EdgeSelectorFunc
}

func (c *testContract) ID() string                     { return c.id }
func (c *testContract) RouteSelector() RouteSelector   { return c.sel }
func (c *testContract) EdgeSelector() EdgeSelectorFunc { return c.edgeSel }
func (c *testContract) Encoding() Encoding             { return c.enc }
func (c *testContract) Input() Message                 { return c.input }
func (c *testContract) Output() Message                { return c.output }
func (c *testContract) Handlers() []HandlerFunc        { return c.handlers }
func (c *testContract) Modifiers() []ModifierFunc      { return c.modifiers }

type testService struct {
	name      string
	contracts []Contract
}

func (s testService) Name() string          { return s.name }
func (s testService) Contracts() []Contract { return s.contracts }

type testServiceBuilder struct {
	svc Service
}

func (b testServiceBuilder) Build() Service { return b.svc }

type testTracer struct {
	injectCalls  int
	extractCalls int
	handlerCalls int
}

func (t *testTracer) Inject(context.Context, TraceCarrier) {
	t.injectCalls++
}

func (t *testTracer) Extract(ctx context.Context, _ TraceCarrier) context.Context {
	t.extractCalls++

	return ctx
}

func (t *testTracer) Handler() HandlerFunc {
	return func(ctx *Context) {
		t.handlerCalls++
		ctx.Next()
	}
}

type testConnDelegate struct {
	opened []Conn
	closed []uint64
}

func (d *testConnDelegate) OnOpen(c Conn) {
	d.opened = append(d.opened, c)
}

func (d *testConnDelegate) OnClose(id uint64) {
	d.closed = append(d.closed, id)
}
