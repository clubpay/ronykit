package stub

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	"github.com/clubpay/ronykit/kit/utils/reflector"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

var defaultConcurrency = 1024 * runtime.NumCPU()

type Stub struct {
	cfg config
	r   *reflector.Reflector

	httpC *fasthttp.Client
}

func New(hostPort string, opts ...Option) *Stub {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}

	cfg := config{
		name:         "RonyKIT Client",
		hostPort:     hostPort,
		readTimeout:  time.Minute * 5,
		writeTimeout: time.Minute * 5,
		dialTimeout:  time.Second * 45,
		l:            common.NewNopLogger(),
		codec:        kit.GetMessageCodec(),
		rootCAs:      rootCAs,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	httpC := &fasthttp.Client{
		Name:                          cfg.name,
		ReadTimeout:                   cfg.readTimeout,
		WriteTimeout:                  cfg.writeTimeout,
		DisableHeaderNamesNormalizing: true,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: cfg.skipVerifyTLS, //nolint:gosec
			RootCAs:            rootCAs,
		},
	}

	if cfg.dialFunc != nil {
		httpC.Dial = cfg.dialFunc
	}

	return &Stub{
		cfg:   cfg,
		r:     reflector.New(),
		httpC: httpC,
	}
}

func HTTP(rawURL string, opts ...Option) (*RESTCtx, error) {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	default:
		return nil, errUnsupportedScheme
	case "http":
	case "https":
		opts = append(opts, Secure())
	}

	s := New(u.Host, opts...).REST()
	s.SetPath(u.Path)

	for k, v := range u.Query() {
		for _, vv := range v {
			s.AppendQuery(k, vv)
		}
	}

	return s, nil
}

func (s *Stub) REST(opt ...RESTOption) *RESTCtx {
	ctx := &RESTCtx{
		c:        s.httpC,
		r:        s.r,
		handlers: map[int]RESTResponseHandler{},
		uri:      fasthttp.AcquireURI(),
		args:     fasthttp.AcquireArgs(),
		req:      fasthttp.AcquireRequest(),
		res:      fasthttp.AcquireResponse(),
		timeout:  s.cfg.readTimeout,
		codec:    s.cfg.codec,
	}

	if s.cfg.secure {
		ctx.uri.SetScheme("https")
	} else {
		ctx.uri.SetScheme("http")
	}

	ctx.cfg.tp = s.cfg.tp
	ctx.uri.SetHost(s.cfg.hostPort)
	ctx.DumpRequestTo(s.cfg.dumpReq)
	ctx.DumpResponseTo(s.cfg.dumpRes)

	for _, o := range opt {
		o(&ctx.cfg)
	}

	return ctx
}

func (s *Stub) Websocket(opts ...WebsocketOption) *WebsocketCtx {
	defaultProxy := http.ProxyFromEnvironment
	if s.cfg.proxy != nil {
		defaultProxy = func(req *http.Request) (*url.URL, error) {
			return s.cfg.proxy.ProxyFunc()(req.URL)
		}
	}

	defaultDialerBuilder := func() *websocket.Dialer {
		return &websocket.Dialer{
			Proxy:            defaultProxy,
			HandshakeTimeout: s.cfg.dialTimeout,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: s.cfg.skipVerifyTLS, //nolint:gosec
				RootCAs:            s.cfg.rootCAs,
			},
		}
	}
	ctx := &WebsocketCtx{
		cfg: wsConfig{
			autoReconnect:   true,
			pingTime:        time.Second * 30,
			dialTimeout:     s.cfg.dialTimeout,
			writeTimeout:    s.cfg.writeTimeout,
			rateLimitChan:   make(chan struct{}, defaultConcurrency),
			rpcInFactory:    common.SimpleIncomingJSONRPC,
			rpcOutFactory:   common.SimpleOutgoingJSONRPC,
			dialerBuilder:   defaultDialerBuilder,
			tracePropagator: s.cfg.tp,
		},
		r:              s.r,
		l:              s.cfg.l,
		pending:        make(map[string]chan kit.IncomingRPCContainer, 1024),
		disconnectChan: make(chan struct{}, 1),
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

var errUnsupportedScheme = errors.New("unsupported scheme")
