package fasthttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/fasthttp/websocket"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/utils"
	"github.com/valyala/fasthttp"
)

const (
	queryMethod    = "fasthttp.Method"
	queryPath      = "fasthttp.path"
	queryDecoder   = "fasthttp.decoder"
	queryPredicate = "fasthttp.predicate"
)

type bundle struct {
	d        ronykit.GatewayDelegate
	srv      *fasthttp.Server
	listen   string
	connPool sync.Pool
	cors     *cors

	httpMux *mux

	wsRoutes     map[string]*routeData
	wsEndpoint   string
	predicateKey string
}

func New(opts ...Option) (*bundle, error) {
	r := &bundle{
		httpMux: &mux{
			RedirectTrailingSlash:  true,
			RedirectFixedPath:      true,
			HandleMethodNotAllowed: true,
			HandleOPTIONS:          true,
		},
		wsRoutes: map[string]*routeData{},
		srv:      &fasthttp.Server{},
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
	_ = (&websocket.FastHTTPUpgrader{}).Upgrade(ctx,
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
				go b.d.OnMessage(wsc, in)
			}
			b.d.OnClose(wsc.id)
		},
	)
}

func (b *bundle) Register(svc ronykit.Service) {
	for _, contract := range svc.Contracts() {
		var h []ronykit.Handler
		h = append(h, svc.PreHandlers()...)
		h = append(h, contract.Handlers()...)
		h = append(h, svc.PostHandlers()...)

		b.registerRPC(svc.Name(), contract, h...)
		b.registerREST(svc.Name(), contract, h...)
	}
}

func (b *bundle) registerRPC(svcName string, c ronykit.Contract, handlers ...ronykit.Handler) {
	rpcSelector, ok := c.Selector().(ronykit.RPCRouteSelector)
	if !ok {
		return
	}

	rd := &routeData{
		ServiceName: svcName,
		Predicate:   rpcSelector.GetPredicate(),
		Handlers:    handlers,
		Modifiers:   c.Modifiers(),
		Factory:     ronykit.CreateMessageFactory(c.Input()),
	}

	b.wsRoutes[rd.Predicate] = rd
}

func (b *bundle) registerREST(svcName string, c ronykit.Contract, handlers ...ronykit.Handler) {
	restSelector, ok := c.Selector().(ronykit.RESTRouteSelector)
	if !ok {
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
		},
	)
}

func (b *bundle) Dispatch(c ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	switch c := c.(type) {
	case *httpConn:
		return b.dispatchHTTP(c, in)
	case *wsConn:
		return b.dispatchWS(in)
	default:
		panic("BUG!! incorrect connection")
	}
}

func (b *bundle) dispatchWS(in []byte) (ronykit.DispatchFunc, error) {
	inputMsgContainer := &incomingMessage{}
	err := json.Unmarshal(in, inputMsgContainer)
	if err != nil {
		return nil, err
	}

	routeData := b.wsRoutes[inputMsgContainer.Header[b.predicateKey]]
	if routeData.Handlers == nil {
		return nil, errNoHandler
	}

	msg := routeData.Factory()
	err = inputMsgContainer.Unmarshal(msg)
	if err != nil {
		return nil, err
	}

	writeFunc := func(conn ronykit.Conn, e *ronykit.Envelope) error {
		for idx := range routeData.Modifiers {
			routeData.Modifiers[idx](e)
		}

		outputMsgContainer := acquireOutgoingMessage()
		outputMsgContainer.Payload = e.GetMsg()
		e.WalkHdr(func(key string, val string) bool {
			outputMsgContainer.Header[key] = val

			return true
		})

		data, err := outputMsgContainer.Marshal()
		if err != nil {
			return err
		}

		_, err = conn.Write(data)

		releaseOutgoingMessage(outputMsgContainer)

		return err
	}

	// return the DispatchFunc
	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		ctx.In().
			SetHdrMap(inputMsgContainer.Header).
			SetMsg(msg)

		ctx.
			Set(ronykit.CtxServiceName, routeData.ServiceName).
			Set(ronykit.CtxRoute, routeData.Predicate)

		// run the execFunc with generated params
		execFunc(writeFunc, routeData.Handlers...)

		return nil
	}, nil
}

func (b *bundle) dispatchHTTP(conn *httpConn, in []byte) (ronykit.DispatchFunc, error) {
	routeData, params, _ := b.httpMux.Lookup(conn.GetMethod(), conn.GetPath())

	// check CORS rules
	b.cors.handle(conn, routeData != nil)

	if routeData == nil {
		return nil, errRouteNotFound
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

	// Set the write function which
	writeFunc := func(c ronykit.Conn, e *ronykit.Envelope) error {
		rc, ok := c.(*httpConn)
		if !ok {
			panic("BUG!! incorrect connection")
		}

		for idx := range routeData.Modifiers {
			routeData.Modifiers[idx](e)
		}

		data, err := e.GetMsg().Marshal()
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

	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		// Set the route and service name
		ctx.Set(ronykit.CtxServiceName, routeData.ServiceName)
		ctx.Set(ronykit.CtxRoute, fmt.Sprintf("%s %s", routeData.Method, routeData.Path))

		ctx.In().
			SetHdrWalker(conn).
			SetMsg(routeData.Decoder(params, in))

		// execute handler functions
		execFunc(writeFunc, routeData.Handlers...)

		return nil
	}, nil
}

func (b *bundle) Start() {
	ln, err := net.Listen("tcp4", b.listen)
	if err != nil {
		panic(err)
	}
	go func() {
		err := b.srv.Serve(ln)
		if err != nil {
			panic(err)
		}
	}()
}

func (b *bundle) Shutdown() {
	_ = b.srv.Shutdown()
}

func (b *bundle) Subscribe(d ronykit.GatewayDelegate) {
	b.d = d
}

var (
	errRouteNotFound = fmt.Errorf("route not found")
	errNoHandler     = fmt.Errorf("no handler for request")
)
