package rest

import (
	"net"

	"github.com/ronaksoft/ronykit"
	"github.com/valyala/fasthttp"
)

type config struct {
	listen string
}

type gateway struct {
	cfg config
	srv *fasthttp.Server
	d   ronykit.GatewayDelegate
}

func newGateway(cfg config) *gateway {
	gw := &gateway{
		cfg: cfg,
	}
	gw.srv = &fasthttp.Server{
		Handler: gw.handler,
	}

	return gw
}

func (g *gateway) handler(ctx *fasthttp.RequestCtx) {
	conn := &conn{
		ctx: ctx,
	}
	g.d.OnOpen(conn)
	_ = g.d.OnMessage(conn, 0, ctx.PostBody())
	g.d.OnClose(conn.ConnID())
}

func (g *gateway) Start() {
	ln, err := net.Listen("tcp4", g.cfg.listen)
	if err != nil {
		panic(err)
	}
	go func() {
		_ = g.srv.Serve(ln)
	}()
}

func (g *gateway) Shutdown() {
	_ = g.srv.Shutdown()
}

func (g *gateway) Subscribe(d ronykit.GatewayDelegate) {
	g.d = d
}
