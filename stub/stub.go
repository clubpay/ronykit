package stub

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/clubpay/ronykit/utils/reflector"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

type Stub struct {
	cfg config

	httpC fasthttp.Client
	r     *reflector.Reflector
}

func New(hostPort string, opts ...Option) *Stub {
	cfg := config{
		name:         "RonyKIT Client",
		hostPort:     hostPort,
		readTimeout:  time.Minute * 5,
		writeTimeout: time.Minute * 5,
		dialTimeout:  time.Second * 45,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Stub{
		cfg: cfg,
		r:   reflector.New(),
		httpC: fasthttp.Client{
			Name:         cfg.name,
			ReadTimeout:  cfg.readTimeout,
			WriteTimeout: cfg.writeTimeout,
			TLSConfig: &tls.Config{
				InsecureSkipVerify: cfg.skipVerifyTLS,
			},
		},
	}
}

func (s *Stub) REST(opt ...RESTOption) *RESTCtx {
	ctx := &RESTCtx{
		c:        &s.httpC,
		r:        s.r,
		handlers: map[int]RESTResponseHandler{},
		uri:      fasthttp.AcquireURI(),
		args:     fasthttp.AcquireArgs(),
		req:      fasthttp.AcquireRequest(),
		res:      fasthttp.AcquireResponse(),
	}

	if s.cfg.secure {
		ctx.uri.SetScheme("https")
	} else {
		ctx.uri.SetScheme("http")
	}

	ctx.uri.SetHost(s.cfg.hostPort)
	ctx.DumpRequestTo(s.cfg.dumpReq)
	ctx.DumpResponseTo(s.cfg.dumpRes)

	for _, o := range opt {
		o(&ctx.cfg)
	}

	return ctx
}

func (s *Stub) Websocket(opts ...WebsocketOption) *WebsocketCtx {
	ctx := &WebsocketCtx{
		cfg: wsConfig{
			d: &websocket.Dialer{
				Proxy:            http.ProxyFromEnvironment,
				HandshakeTimeout: s.cfg.dialTimeout,
			},
		},
		r:        s.r,
		handlers: map[string]WSResponseHandler{},
	}

	for _, o := range opts {
		o(&ctx.cfg)
	}

	return ctx
}
