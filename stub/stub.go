package stub

import (
	"time"

	"github.com/valyala/fasthttp"
)

type Stub struct {
	cfg config

	httpC fasthttp.Client
}

func New(hostPort string, opts ...Option) *Stub {
	cfg := config{
		hostPort:     hostPort,
		readTimeout:  time.Minute * 5,
		writeTimeout: time.Minute * 5,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Stub{
		cfg: cfg,
		httpC: fasthttp.Client{
			ReadTimeout:  cfg.readTimeout,
			WriteTimeout: cfg.writeTimeout,
		},
	}
}

func (s *Stub) HTTP() *restClientCtx {
	hc := &restClientCtx{
		c:        &s.httpC,
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

	return hc
}
