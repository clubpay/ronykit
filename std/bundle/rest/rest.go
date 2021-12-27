package rest

import (
	"fmt"
	"net"

	"github.com/valyala/fasthttp"

	"github.com/ronaksoft/ronykit/std/bundle/rest/mux"

	"github.com/ronaksoft/ronykit"
)

var (
	ErrRouteNotFound = fmt.Errorf("route not found")
)

type (
	// ParamsSetter is the interface which should be implemented by the
	// params writer for `search` in order to store the found named path parameters, if any.
	ParamsSetter interface {
		Set(string, interface{})
	}
	// ParamsGetter is the interface which is used on the DecoderFunc to get the extracted data
	// from path parameters.
	ParamsGetter interface {
		Get(key string) interface{}
	}
	// DecoderFunc is the function which gets the raw http's body and the extracted data from
	// path and must return a ronykit.Message.
	DecoderFunc func(bag ParamsGetter, data []byte) ronykit.Message
)

type rest struct {
	srv    *fasthttp.Server
	listen string
	d      ronykit.GatewayDelegate
	mux    *mux.Router
}

func New(opts ...Option) (*rest, error) {
	r := &rest{
		mux: mux.New(),
		srv: &fasthttp.Server{},
	}

	for _, opt := range opts {
		opt(r)
	}

	r.srv.Handler = r.handler

	return r, nil
}

func (r rest) Dispatch(conn ronykit.Conn, streamID int64, in []byte) ronykit.DispatchFunc {
	rc, ok := conn.(ronykit.REST)
	if !ok {
		return nil
	}

	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		h, params, _ := r.mux.Lookup(rc.GetMethod(), rc.GetPath())
		if h == nil {
			return ErrRouteNotFound
		}

		writeFunc := func(m ronykit.Message) {
			data, err := m.Marshal()
			if err != nil {
				ctx.Error(err)
			}

			ctx.Walk(
				func(key string, val interface{}) bool {
					if v, ok := val.(string); ok {
						rc.WriteHeader(key, v)
					}

					return true
				},
			)

			ctx.Error(conn.Write(streamID, data))
		}

		execFunc(h.Decoder(params, in), writeFunc, h.Handlers...)

		return nil
	}
}

func (r *rest) Set(method, path string, decoder mux.DecoderFunc, handlers ...ronykit.Handler) {
	r.mux.Handle(
		method,
		path,
		&mux.Handle{
			Decoder:  decoder,
			Handlers: handlers,
		},
	)
}

func (r *rest) handler(ctx *fasthttp.RequestCtx) {
	conn := &conn{
		ctx: ctx,
	}
	r.d.OnOpen(conn)
	_ = r.d.OnMessage(conn, 0, ctx.PostBody())
	r.d.OnClose(conn.ConnID())
}

func (r *rest) Start() {
	ln, err := net.Listen("tcp4", r.listen)
	if err != nil {
		panic(err)
	}
	go func() {
		_ = r.srv.Serve(ln)
	}()
}

func (r *rest) Shutdown() {
	_ = r.srv.Shutdown()
}

func (r *rest) Subscribe(d ronykit.GatewayDelegate) {
	r.d = d
}

func (r *rest) Gateway() ronykit.Gateway {
	return r
}

func (r *rest) Dispatcher() ronykit.Dispatcher {
	return r
}
