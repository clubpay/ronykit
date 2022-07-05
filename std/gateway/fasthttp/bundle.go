package fasthttp

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/internal/common"
	"github.com/clubpay/ronykit/utils"
	"github.com/clubpay/ronykit/utils/buf"
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

var _ ronykit.Gateway = (*bundle)(nil)

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

	r.wsUpgrade.CheckOrigin = func(ctx *fasthttp.RequestCtx) bool {
		return true
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

func (b *bundle) Register(
	svcName, contractID string, enc ronykit.Encoding, sel ronykit.RouteSelector, input ronykit.Message,
) {
	b.registerRPC(svcName, contractID, enc, sel, input)
	b.registerREST(svcName, contractID, enc, sel, input)
}

func (b *bundle) registerRPC(
	svcName, contractID string, enc ronykit.Encoding, sel ronykit.RouteSelector, input ronykit.Message,
) {
	rpcSelector, ok := sel.(ronykit.RPCRouteSelector)
	if !ok {
		// this selector is not an RPCRouteSelector then we return with no
		// extra action taken.
		return
	}

	// We don't accept selector with empty Predicate for the obvious reason.
	if rpcSelector.GetPredicate() == "" {
		return
	}

	rd := &routeData{
		ServiceName: svcName,
		ContractID:  contractID,
		Predicate:   rpcSelector.GetPredicate(),
		Factory:     ronykit.CreateMessageFactory(input),
	}

	b.wsRoutes[rd.Predicate] = rd
}

func (b *bundle) registerREST(
	svcName, contractID string, enc ronykit.Encoding, sel ronykit.RouteSelector, input ronykit.Message,
) {
	restSelector, ok := sel.(ronykit.RESTRouteSelector)
	if !ok {
		return
	}

	if restSelector.GetMethod() == "" || restSelector.GetPath() == "" {
		return
	}

	decoder, ok := restSelector.Query(queryDecoder).(DecoderFunc)
	if !ok || decoder == nil {
		decoder = reflectDecoder(enc, ronykit.CreateMessageFactory(input))
	}

	b.httpMux.Handle(
		restSelector.GetMethod(), restSelector.GetPath(),
		&routeData{
			ServiceName: svcName,
			ContractID:  contractID,
			Method:      restSelector.GetMethod(),
			Path:        restSelector.GetPath(),
			Decoder:     decoder,
		},
	)
}

func (b *bundle) Dispatch(ctx *ronykit.Context, in []byte) (ronykit.ExecuteArg, error) {
	switch ctx.Conn().(type) {
	case *httpConn:
		return b.httpDispatch(ctx, in)
	case *wsConn:
		return b.wsDispatch(ctx, in)
	default:
		panic("BUG!! incorrect connection")
	}
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

				inBuf := buf.FromBytes(in)
				go b.wsHandlerExec(inBuf, wsc)
			}
			wsc.Close()
			b.d.OnClose(wsc.id)
		},
	)
}

func (b *bundle) wsHandlerExec(buf *buf.Bytes, wsc *wsConn) {
	b.d.OnMessage(wsc, *buf.Bytes())
	buf.Release()
}

func (b *bundle) wsDispatch(ctx *ronykit.Context, in []byte) (ronykit.ExecuteArg, error) {
	inputMsgContainer := b.rpcInFactory()
	err := inputMsgContainer.Unmarshal(in)
	if err != nil {
		return ronykit.NoExecuteArg, err
	}

	routeData := b.wsRoutes[inputMsgContainer.GetHdr(b.predicateKey)]
	if routeData == nil {
		return ronykit.NoExecuteArg, errNoHandler
	}

	msg := routeData.Factory()
	err = inputMsgContainer.Fill(msg)
	if err != nil {
		return ronykit.NoExecuteArg, err
	}

	ctx.In().
		SetHdrMap(inputMsgContainer.GetHdrMap()).
		SetMsg(msg)

	return ronykit.ExecuteArg{
		WriteFunc:   b.wsWriteFunc,
		ServiceName: routeData.ServiceName,
		ContractID:  routeData.ContractID,
		Route:       routeData.Predicate,
	}, nil
}

func (b *bundle) wsWriteFunc(conn ronykit.Conn, e ronykit.Envelope) error {
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

func (b *bundle) httpDispatch(ctx *ronykit.Context, in []byte) (ronykit.ExecuteArg, error) {
	//nolint:forcetypeassert
	conn := ctx.Conn().(*httpConn)

	routeData, params, _ := b.httpMux.Lookup(conn.GetMethod(), conn.GetPath())

	// check CORS rules before even returning errRouteNotFound. This makes sure that
	// we handle any CORS even for non-routable requests.
	b.cors.handle(conn, routeData != nil)

	if routeData == nil {
		return ronykit.NoExecuteArg, errRouteNotFound
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

	m, err := routeData.Decoder(params, in)
	if err != nil {
		return ronykit.NoExecuteArg, err
	}

	ctx.In().
		SetHdrWalker(conn).
		SetMsg(m)

	return ronykit.ExecuteArg{
		WriteFunc:   b.httpWriteFunc,
		ServiceName: routeData.ServiceName,
		ContractID:  routeData.ContractID,
		Route:       fmt.Sprintf("%s %s", routeData.Method, routeData.Path),
	}, nil
}

func (b *bundle) httpWriteFunc(c ronykit.Conn, e ronykit.Envelope) error {
	rc, ok := c.(*httpConn)
	if !ok {
		panic("BUG!! incorrect connection")
	}

	var (
		data []byte
		err  error
	)

	data, err = ronykit.MarshalMessage(e.GetMsg())
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
	errRouteNotFound    = fmt.Errorf("route not found")
	errNoHandler        = fmt.Errorf("no handler for request")
	errConnectionClosed = fmt.Errorf("connection closed")
)
