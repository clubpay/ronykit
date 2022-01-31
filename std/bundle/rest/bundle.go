package rest

import (
	"fmt"
	"net"
	"sync"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest/mux"
	"github.com/ronaksoft/ronykit/utils"
	"github.com/valyala/fasthttp"
)

const (
	queryMethod  = "rest.method"
	queryPath    = "rest.path"
	queryDecoder = "rest.decoder"
)

type bundle struct {
	srv      *fasthttp.Server
	listen   string
	d        ronykit.GatewayDelegate
	mux      *mux.Router
	connPool sync.Pool
}

func New(opts ...Option) (*bundle, error) {
	r := &bundle{
		mux: mux.New(),
		srv: &fasthttp.Server{},
	}

	for _, opt := range opts {
		opt(r)
	}

	r.srv.Handler = r.handler

	return r, nil
}

func MustNew(opts ...Option) *bundle {
	b, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return b
}

func (r *bundle) handler(ctx *fasthttp.RequestCtx) {
	c, ok := r.connPool.Get().(*conn)
	if !ok {
		c = &conn{}
	}

	c.ctx = ctx
	r.d.OnOpen(c)
	r.d.OnMessage(c, ctx.PostBody())
	r.d.OnClose(c.ConnID())

	c.reset()
	r.connPool.Put(c)
}

func (r *bundle) Register(svc ronykit.Service) {
	for _, rt := range svc.Contracts() {
		var h []ronykit.Handler
		h = append(h, svc.PreHandlers()...)
		h = append(h, rt.Handlers()...)
		h = append(h, svc.PostHandlers()...)

		method, ok := rt.Query(queryMethod).(string)
		if !ok {
			panic("method is not set in Service's Contract")
		}
		path, ok := rt.Query(queryPath).(string)
		if !ok {
			panic("path is not set in Service's Contract")
		}
		decoder, ok := rt.Query(queryDecoder).(mux.DecoderFunc)
		if !ok {
			panic("mux.DecoderFunc is not set in Service's Contract")
		}

		r.mux.Handle(
			method, path,
			&mux.RouteData{
				ServiceName: svc.Name(),
				Decoder:     decoder,
				Handlers:    h,
				Modifiers:   rt.Modifiers(),
			},
		)
	}
}

func (r *bundle) Dispatch(c ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	rc, ok := c.(*conn)
	if !ok {
		panic("BUG!! incorrect connection")
	}

	routeData, params, _ := r.mux.Lookup(rc.GetMethod(), rc.GetPath())
	if routeData == nil {
		return nil, errRouteNotFound
	}

	// Walk over all the query params
	rc.ctx.QueryArgs().VisitAll(
		func(key, value []byte) {
			params = append(
				params,
				mux.Param{
					Key:   utils.B2S(key),
					Value: utils.B2S(value),
				},
			)
		},
	)

	// Set the write function which
	writeFunc := func(c ronykit.Conn, e *ronykit.Envelope) error {
		rc, ok := c.(*conn)
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
		ctx.Set(ronykit.CtxRoute, fmt.Sprintf("%s %s", rc.GetMethod(), rc.GetPath()))

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
	errNonRestConnection = fmt.Errorf("incompatible connection, expected REST conn")
	errRouteNotFound     = fmt.Errorf("route not found")
)
