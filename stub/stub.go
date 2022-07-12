package stub

import (
	"crypto/tls"
	"time"

	"github.com/clubpay/ronykit/utils/reflector"
	"github.com/valyala/fasthttp"
)

type Stub struct {
	cfg config

	httpC fasthttp.Client
	r     *reflector.Reflector
}

func New(hostPort string, opts ...Option) *Stub {
	cfg := config{
		name:         "RonyKIT StubClient",
		hostPort:     hostPort,
		readTimeout:  time.Minute * 5,
		writeTimeout: time.Minute * 5,
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

func (s *Stub) REST() *restClientCtx {
	hc := &restClientCtx{
		c:        &s.httpC,
		r:        s.r,
		handlers: map[int]RESTResponseHandler{},
		uri:      fasthttp.AcquireURI(),
		args:     fasthttp.AcquireArgs(),
		req:      fasthttp.AcquireRequest(),
		res:      fasthttp.AcquireResponse(),
	}

	if s.cfg.secure {
		hc.uri.SetScheme("https")
	} else {
		hc.uri.SetScheme("http")
	}

	hc.uri.SetHost(s.cfg.hostPort)
	hc.DumpRequestTo(s.cfg.dumpReq)
	hc.DumpResponseTo(s.cfg.dumpRes)

	return hc
}
