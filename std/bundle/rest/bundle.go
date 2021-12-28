package rest

import (
	"fmt"
	"net"
	"sync"

	"github.com/valyala/fasthttp"

	"github.com/ronaksoft/ronykit/std/bundle/rest/mux"

	"github.com/ronaksoft/ronykit"
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

func (r *bundle) Dispatch(conn ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	rc, ok := conn.(ronykit.REST)
	if !ok {
		return nil, errNoHandler
	}

	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		h, params, _ := r.mux.Lookup(rc.GetMethod(), rc.GetPath())
		if h == nil {
			return errRouteNotFound
		}

		writeFunc := func(m ronykit.Message) {
			data, err := m.Marshal()
			if err != nil {
				ctx.Error(err)
			}

			ctx.Walk(
				func(key string, val interface{}) bool {
					if v, ok := val.(string); ok {
						rc.Set(key, v)
					}

					return true
				},
			)

			_, err = conn.Write(data)
			ctx.Error(err)
		}

		execFunc(h.Decoder(params, in), writeFunc, h.Handlers...)

		return nil
	}, nil
}

func (r *bundle) Set(method, path string, decoder mux.DecoderFunc, handlers ...ronykit.Handler) {
	r.mux.Handle(
		method,
		path,
		&mux.Handle{
			Decoder:  decoder,
			Handlers: handlers,
		},
	)
}

func (r *bundle) handler(ctx *fasthttp.RequestCtx) {
	c, ok := r.connPool.Get().(*conn)
	if !ok {
		c = &conn{}
	}

	c.ctx = ctx
	r.d.OnOpen(c)
	_ = r.d.OnMessage(c, ctx.PostBody())
	r.d.OnClose(c.ConnID())

	c.reset()
	r.connPool.Put(c)
}

func (r *bundle) Start() {
	ln, err := net.Listen("tcp4", r.listen)
	if err != nil {
		panic(err)
	}
	go func() {
		_ = r.srv.Serve(ln)
	}()
}

func (r *bundle) Shutdown() {
	_ = r.srv.Shutdown()
}

func (r *bundle) Subscribe(d ronykit.GatewayDelegate) {
	r.d = d
}

func (r *bundle) Gateway() ronykit.Gateway {
	return r
}

func (r *bundle) Dispatcher() ronykit.Dispatcher {
	return r
}

var (
	errRouteNotFound = fmt.Errorf("route not found")
	errNoHandler     = fmt.Errorf("no handler")
)
