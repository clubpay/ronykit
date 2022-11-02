package stub

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	"github.com/clubpay/ronykit/kit/utils/reflector"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

type (
	Header              map[string]string
	RPCContainerHandler func(c kit.IncomingRPCContainer)
	RPCMessageHandler   func(ctx context.Context, msg kit.Message, hdr Header, err error)
)

type Stub struct {
	cfg config
	r   *reflector.Reflector

	httpC fasthttp.Client
}

func New(hostPort string, opts ...Option) *Stub {
	cfg := config{
		name:         "RonyKIT Client",
		hostPort:     hostPort,
		readTimeout:  time.Minute * 5,
		writeTimeout: time.Minute * 5,
		dialTimeout:  time.Second * 45,
		l:            nopLogger{},
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
			autoReconnect: true,
			pingTime:      time.Second * 30,
			dialTimeout:   s.cfg.dialTimeout,
			writeTimeout:  s.cfg.writeTimeout,
			rpcInFactory:  common.SimpleIncomingJSONRPC,
			rpcOutFactory: common.SimpleOutgoingJSONRPC,
			dialerBuilder: func() *websocket.Dialer {
				return &websocket.Dialer{
					Proxy:            http.ProxyFromEnvironment,
					HandshakeTimeout: s.cfg.dialTimeout,
				}
			},
		},
		r:       s.r,
		l:       s.cfg.l,
		pending: make(map[string]chan kit.IncomingRPCContainer, 1024),
	}

	for _, o := range opts {
		o(&ctx.cfg)
	}

	if s.cfg.secure {
		ctx.url = fmt.Sprintf("wss://%s", s.cfg.hostPort)
	} else {
		ctx.url = fmt.Sprintf("ws://%s", s.cfg.hostPort)
	}

	return ctx
}
