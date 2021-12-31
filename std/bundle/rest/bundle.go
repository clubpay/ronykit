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

func (r *bundle) SetHandler(
	method, path string, decoder mux.DecoderFunc, handlers ...ronykit.Handler,
) {
	r.mux.Handle(
		method,
		path,
		&mux.Handle{
			Decoder:  decoder,
			Handlers: handlers,
		},
	)
}

func (r *bundle) Dispatch(c ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	rc, ok := c.(*conn)
	if !ok {
		panic("BUG!! incorrect connection")
	}

	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		h, params, _ := r.mux.Lookup(rc.GetMethod(), rc.GetPath())
		if h == nil {
			return errNonRestConnection
		}

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

		writeFunc := func(m ronykit.Message, ctxKeys ...string) error {
			data, err := m.Marshal()
			if err != nil {
				return err
			}

			for idx := range ctxKeys {
				if v, ok := ctx.Get(ctxKeys[idx]).(string); ok {
					rc.ctx.Response.Header.Set(ctxKeys[idx], v)
				}
			}

			rc.ctx.SetBody(data)

			return nil
		}

		execFunc(h.Decoder(params, in), writeFunc, h.Handlers...)

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
)
