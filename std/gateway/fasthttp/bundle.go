package fasthttp

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/internal/common"
	"github.com/clubpay/ronykit/utils"
	"github.com/clubpay/ronykit/utils/pools"
	"github.com/clubpay/ronykit/utils/pools/buf"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

const (
	queryMethod    = "fasthttp.Method"
	queryPath      = "fasthttp.path"
	queryDecoder   = "fasthttp.decoder"
	queryPredicate = "fasthttp.predicate"
)

type bundle struct {
	l        ronykit.Logger
	d        ronykit.GatewayDelegate
	srv      *fasthttp.Server
	listen   string
	connPool sync.Pool
	cors     *cors

	httpMux *mux

	wsUpgrade     websocket.FastHTTPUpgrader
	wsRoutes      map[string]*routeData
	wsEndpoint    string
	predicateKey  string
	rpcInFactory  ronykit.IncomingRPCFactory
	rpcOutFactory ronykit.OutgoingRPCFactory
}

func New(opts ...Option) (*bundle, error) {
	r := &bundle{
		httpMux: &mux{
			RedirectTrailingSlash:  true,
			RedirectFixedPath:      true,
			HandleMethodNotAllowed: true,
			HandleOPTIONS:          true,
		},
		wsRoutes:      map[string]*routeData{},
		srv:           &fasthttp.Server{},
		rpcInFactory:  common.SimpleIncomingJSONRPC,
		rpcOutFactory: common.SimpleOutgoingJSONRPC,
		l:             common.NewNopLogger(),
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.wsEndpoint != "" {
		wsEndpoint := utils.S2B(r.wsEndpoint)
		r.srv.Handler = func(ctx *fasthttp.RequestCtx) {
			if ctx.IsGet() && bytes.EqualFold(ctx.Path(), wsEndpoint) {
				r.wsHandler(ctx)
			} else {
				r.httpHandler(ctx)
			}
		}
	} else {
		r.srv.Handler = r.httpHandler
	}

	return r, nil
}

func MustNew(opts ...Option) *bundle {
	b, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return b
}

func (b *bundle) httpHandler(ctx *fasthttp.RequestCtx) {
	c, ok := b.connPool.Get().(*httpConn)
	if !ok {
		c = &httpConn{}
	}

	c.ctx = ctx
	b.d.OnOpen(c)
	b.d.OnMessage(c, ctx.PostBody())
	b.d.OnClose(c.ConnID())

	b.connPool.Put(c)
}

func (b *bundle) wsHandler(ctx *fasthttp.RequestCtx) {
	_ = b.wsUpgrade.Upgrade(ctx,
		func(conn *websocket.Conn) {
			wsc := &wsConn{
				kv:       map[string]string{},
				id:       0,
				clientIP: conn.RemoteAddr().String(),
				c:        conn,
			}
			b.d.OnOpen(wsc)
			for {
				_, in, err := conn.ReadMessage()
				if err != nil {
					break
				}

				inBuf := pools.Buffer.FromBytes(in)
				go func(buf *buf.Bytes) {
					b.d.OnMessage(wsc, *buf.Bytes())
					pools.Buffer.Put(buf)
				}(inBuf)
			}
			b.d.OnClose(wsc.id)
		},
	)
}

func (b *bundle) Register(svc ronykit.Service) {
	for _, contract := range svc.Contracts() {
		b.registerRPC(svc.Name(), contract, contract.Handlers()...)
		b.registerREST(svc.Name(), contract, contract.Handlers()...)
	}
}

func (b *bundle) registerRPC(svcName string, c ronykit.Contract, handlers ...ronykit.HandlerFunc) {
	rpcSelector, ok := c.RouteSelector().(ronykit.RPCRouteSelector)
	if !ok {
		return
	}

	if rpcSelector.GetPredicate() == "" {
		return
	}

	rd := &routeData{
		ServiceName: svcName,
		Predicate:   rpcSelector.GetPredicate(),
		Handlers:    handlers,
		Modifiers:   c.Modifiers(),
		Factory:     ronykit.CreateMessageFactory(c.Input()),
		Encoding:    c.Encoding(),
	}

	b.wsRoutes[rd.Predicate] = rd
}

func (b *bundle) registerREST(svcName string, c ronykit.Contract, handlers ...ronykit.HandlerFunc) {
	restSelector, ok := c.RouteSelector().(ronykit.RESTRouteSelector)
	if !ok {
		return
	}

	if restSelector.GetMethod() == "" || restSelector.GetPath() == "" {
		return
	}

	decoder, ok := restSelector.Query(queryDecoder).(DecoderFunc)
	if !ok || decoder == nil {
		decoder = reflectDecoder(ronykit.CreateMessageFactory(c.Input()))
	}

	b.httpMux.Handle(
		restSelector.GetMethod(), restSelector.GetPath(),
		&routeData{
			ServiceName: svcName,
			Handlers:    handlers,
			Modifiers:   c.Modifiers(),
			Method:      restSelector.GetMethod(),
			Path:        restSelector.GetPath(),
			Decoder:     decoder,
			Encoding:    c.Encoding(),
		},
	)
}

func (b *bundle) Dispatch(ctx *ronykit.Context, in []byte, execFunc ronykit.ExecuteFunc) error {
	switch ctx.Conn().(type) {
	case *httpConn:
		return b.httpDispatch(ctx, in, execFunc)
	case *wsConn:
		return b.wsDispatch(ctx, in, execFunc)
	default:
		panic("BUG!! incorrect connection")
	}
}

func (b *bundle) wsDispatch(ctx *ronykit.Context, in []byte, execFunc ronykit.ExecuteFunc) error {
	inputMsgContainer := b.rpcInFactory()
	err := inputMsgContainer.Unmarshal(in)
	if err != nil {
		return err
	}

	routeData := b.wsRoutes[inputMsgContainer.GetHdr(b.predicateKey)]
	if routeData.Handlers == nil {
		return errNoHandler
	}

	msg := routeData.Factory()
	err = inputMsgContainer.Fill(msg)
	if err != nil {
		return err
	}

	ctx.In().
		SetHdrMap(inputMsgContainer.GetHdrMap()).
		SetMsg(msg)

	ctx.
		Set(ronykit.CtxServiceName, routeData.ServiceName).
		Set(ronykit.CtxRoute, routeData.Predicate)

	ctx.AddModifier(routeData.Modifiers...)

	// run the execFunc with generated params
	execFunc(ctx, b.wsWriteFunc, routeData.Handlers...)

	return nil
}

func (b *bundle) wsWriteFunc(conn ronykit.Conn, e *ronykit.Envelope) error {
	outputMsgContainer := b.rpcOutFactory()
	outputMsgContainer.SetPayload(e.GetMsg())
	e.WalkHdr(func(key string, val string) bool {
		outputMsgContainer.SetHdr(key, val)

		return true
	})

	data, err := outputMsgContainer.Marshal()
	if err != nil {
		return err
	}

	_, err = conn.Write(data)

	return err
}

func (b *bundle) httpDispatch(ctx *ronykit.Context, in []byte, execFunc ronykit.ExecuteFunc) error {
	//nolint:forcetypeassert
	conn := ctx.Conn().(*httpConn)

	routeData, params, _ := b.httpMux.Lookup(conn.GetMethod(), conn.GetPath())

	// check CORS rules before even returning errRouteNotFound. This makes sure that
	// we handle any CORS even for non-routable requests.
	b.cors.handle(conn, routeData != nil)

	if routeData == nil {
		return errRouteNotFound
	}

	// Walk over all the query params
	conn.ctx.QueryArgs().VisitAll(
		func(key, value []byte) {
			params = append(
				params,
				Param{
					Key:   utils.B2S(key),
					Value: utils.B2S(value),
				},
			)
		},
	)

	// Set the route and service name
	ctx.Set(ronykit.CtxServiceName, routeData.ServiceName)
	ctx.Set(ronykit.CtxRoute, fmt.Sprintf("%s %s", routeData.Method, routeData.Path))

	ctx.In().
		SetHdrWalker(conn).
		SetMsg(routeData.Decoder(params, in))

	ctx.AddModifier(routeData.Modifiers...)

	// execute handler functions
	execFunc(ctx, b.httpWriteFunc, routeData.Handlers...)

	return nil
}

func (b *bundle) httpWriteFunc(c ronykit.Conn, e *ronykit.Envelope) error {
	rc, ok := c.(*httpConn)
	if !ok {
		panic("BUG!! incorrect connection")
	}

	var (
		data []byte
		err  error
	)

	switch m := e.GetMsg().(type) {
	case ronykit.Marshaller:
		data, err = m.Marshal()
	case json.Marshaler:
		data, err = m.MarshalJSON()
	case encoding.BinaryMarshaler:
		data, err = m.MarshalBinary()
	case encoding.TextMarshaler:
		data, err = m.MarshalText()
	default:
		data, err = json.Marshal(m)
	}
	if err != nil {
		return err
	}

	e.WalkHdr(
		func(key string, val string) bool {
			rc.ctx.Response.Header.Set(key, val)

			return true
		},
	)

	rc.ctx.SetBody(data)

	return nil
}

func (b *bundle) Start(_ context.Context) error {
	ln, err := net.Listen("tcp4", b.listen)
	if err != nil {
		return err
	}

	go func() {
		err = b.srv.Serve(ln)
		if err != nil {
			b.l.Errorf("got error on serving: %v", err)
			panic(err)
		}
	}()

	return nil
}

func (b *bundle) Shutdown(_ context.Context) error {
	return b.srv.Shutdown()
}

func (b *bundle) Subscribe(d ronykit.GatewayDelegate) {
	b.d = d
}

var (
	errRouteNotFound = fmt.Errorf("route not found")
	errNoHandler     = fmt.Errorf("no handler for request")
)
