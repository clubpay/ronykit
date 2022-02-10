package fasthttp

import (
	"fmt"
	"net"
	"sync"

	"github.com/fasthttp/router"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/utils"
	"github.com/valyala/fasthttp"
)

const (
	queryMethod  = "fasthttp.method"
	queryPath    = "fasthttp.path"
	queryDecoder = "fasthttp.decoder"
)

type bundle struct {
	srv            *fasthttp.Server
	listen         string
	d              ronykit.GatewayDelegate
	mux            *mux
	connPool       sync.Pool
	cors           *cors
	websocketRoute string
}

func New(opts ...Option) (*bundle, error) {
	r := &bundle{
		mux: &mux{
			RedirectTrailingSlash:  true,
			RedirectFixedPath:      true,
			HandleMethodNotAllowed: true,
			HandleOPTIONS:          true,
		},
		srv: &fasthttp.Server{},
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.websocketRoute != "" {
		entryRouter := router.New()
		entryRouter.GET(r.websocketRoute, r.wsHandler)
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

func (r *bundle) httpHandler(ctx *fasthttp.RequestCtx) {
	c, ok := r.connPool.Get().(*httpConn)
	if !ok {
		c = &httpConn{}
	}

	c.ctx = ctx
	r.d.OnOpen(c)
	r.d.OnMessage(c, ctx.PostBody())
	r.d.OnClose(c.ConnID())

	c.reset()
	r.connPool.Put(c)
}

func (r *bundle) wsHandler(ctx *fasthttp.RequestCtx) {

}

func (r *bundle) Register(svc ronykit.Service) {
	for _, rt := range svc.Contracts() {
		var h []ronykit.Handler
		h = append(h, svc.PreHandlers()...)
		h = append(h, rt.Handlers()...)
		h = append(h, svc.PostHandlers()...)

		method, ok := rt.Query(queryMethod).(string)
		if !ok {
			continue
		}
		path, ok := rt.Query(queryPath).(string)
		if !ok {
			continue
		}
		decoder, ok := rt.Query(queryDecoder).(DecoderFunc)
		if !ok {
			continue
		}

		r.mux.Handle(
			method, path,
			&routeData{
				Method:      method,
				Path:        path,
				ServiceName: svc.Name(),
				Decoder:     decoder,
				Handlers:    h,
				Modifiers:   rt.Modifiers(),
			},
		)
	}
}

func (r *bundle) Dispatch(c ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	rc, ok := c.(*httpConn)
	if !ok {
		panic("BUG!! incorrect connection")
	}

	routeData, params, _ := r.mux.Lookup(rc.GetMethod(), rc.GetPath())
	if routeData == nil {
		if r.cors != nil {
			// ByPass cors (Cross Origin Resource Sharing) check
			if r.cors.origins == "*" {
				rc.ctx.Response.Header.Set(headerAccessControlAllowOrigin, rc.Get(headerOrigin))
			} else {
				rc.ctx.Response.Header.Set(headerAccessControlAllowOrigin, r.cors.origins)
			}

			if rc.ctx.IsOptions() {
				reqHeaders := rc.ctx.Request.Header.Peek(headerAccessControlRequestHeaders)
				if len(reqHeaders) > 0 {
					rc.ctx.Response.Header.SetBytesV(headerAccessControlAllowHeaders, reqHeaders)
				} else {
					rc.ctx.Response.Header.Set(headerAccessControlAllowHeaders, r.cors.headers)
				}

				rc.ctx.Response.Header.Set(headerAccessControlAllowMethods, r.cors.methods)
				rc.ctx.SetStatusCode(fasthttp.StatusNoContent)
			} else {
				rc.ctx.SetStatusCode(fasthttp.StatusNotImplemented)
			}
		}

		return nil, errRouteNotFound
	}

	if r.cors != nil {
		// ByPass cors (Cross Origin Resource Sharing) check
		if r.cors.origins == "*" {
			rc.ctx.Response.Header.Set(headerAccessControlAllowOrigin, rc.Get(headerOrigin))
		} else {
			rc.ctx.Response.Header.Set(headerAccessControlAllowOrigin, r.cors.origins)
		}
	}

	// Walk over all the query params
	rc.ctx.QueryArgs().VisitAll(
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
		// Walk over all the connection headers
		rc.Walk(
			func(key string, val string) bool {
				ctx.In().SetHdr(key, val)

				return true
			},
		)

		// Set the route and service name
		ctx.Set(ronykit.CtxServiceName, routeData.ServiceName)
		ctx.Set(ronykit.CtxRoute, fmt.Sprintf("%s %s", routeData.Method, routeData.Path))

		ctx.In().SetMsg(routeData.Decoder(params, in))
		execFunc(writeFunc, routeData.Handlers...)

		return nil
	}, nil
}

func (r *bundle) Start() {
	ln, err := net.Listen("tcp4", r.listen)
	if err != nil {
		panic(err)
	}
	go func() {
		err := r.srv.Serve(ln)
		if err != nil {
			panic(err)
		}
	}()
}

func (r *bundle) Shutdown() {
	_ = r.srv.Shutdown()
}

func (r *bundle) Subscribe(d ronykit.GatewayDelegate) {
	r.d = d
}

var (
	errRouteNotFound = fmt.Errorf("route not found")
)
