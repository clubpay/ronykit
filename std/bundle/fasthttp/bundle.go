package fasthttp

import (
	"fmt"
	"net"
	"sync"

	"github.com/fasthttp/router"
	"github.com/goccy/go-json"
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
	predicateKey string
	wsEndpoint   string
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
		entryRouter := router.New()
		entryRouter.GET(r.wsEndpoint, r.wsHandler)
		entryRouter.Handle(router.MethodWild, "/", r.httpHandler)
		r.srv.Handler = entryRouter.Handler
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

	c.reset()
	b.connPool.Put(c)
}

func (b *bundle) wsHandler(ctx *fasthttp.RequestCtx) {}

func (b *bundle) Register(svc ronykit.Service) {
	for _, contract := range svc.Contracts() {
		var h []ronykit.Handler
		h = append(h, svc.PreHandlers()...)
		h = append(h, contract.Handlers()...)
		h = append(h, svc.PostHandlers()...)

		b.registerRPC(svc.Name(), contract)
		b.registerREST(svc.Name(), contract)
	}
}

func (b *bundle) registerRPC(svcName string, c ronykit.Contract) {
	rpcSelector, ok := c.Selector().(ronykit.RPCRouteSelector)
	if !ok {
		return
	}

	rd := &routeData{
		ServiceName: svcName,
		Predicate:   rpcSelector.GetPredicate(),
		Handlers:    c.Handlers(),
		Modifiers:   c.Modifiers(),
		Factory:     ronykit.CreateMessageFactory(c.Input()),
	}

	b.wsRoutes[rd.Predicate] = rd
}

func (b *bundle) registerREST(svcName string, c ronykit.Contract) {
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
			Handlers:    c.Handlers(),
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
		return b.dispatchWS(c, in)
	default:
		panic("BUG!! incorrect connection")
	}
}

func (b *bundle) dispatchWS(c *wsConn, in []byte) (ronykit.DispatchFunc, error) {
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
	b.handleCORS(conn, routeData != nil)

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

func (b *bundle) handleCORS(rc *httpConn, routeFound bool) {
	if b.cors == nil {
		return
	}

	// ByPass cors (Cross Origin Resource Sharing) check
	if b.cors.origins == "*" {
		rc.ctx.Response.Header.Set(headerAccessControlAllowOrigin, rc.Get(headerOrigin))
	} else {
		rc.ctx.Response.Header.Set(headerAccessControlAllowOrigin, b.cors.origins)
	}

	if routeFound {
		return
	}

	if rc.ctx.IsOptions() {
		reqHeaders := rc.ctx.Request.Header.Peek(headerAccessControlRequestHeaders)
		if len(reqHeaders) > 0 {
			rc.ctx.Response.Header.SetBytesV(headerAccessControlAllowHeaders, reqHeaders)
		} else {
			rc.ctx.Response.Header.Set(headerAccessControlAllowHeaders, b.cors.headers)
		}

		rc.ctx.Response.Header.Set(headerAccessControlAllowMethods, b.cors.methods)
		rc.ctx.SetStatusCode(fasthttp.StatusNoContent)
	} else {
		rc.ctx.SetStatusCode(fasthttp.StatusNotImplemented)
	}
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
