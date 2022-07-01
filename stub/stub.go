package stub

import (
	"context"
	"time"

	"github.com/clubpay/ronykit"
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

func (s *Stub) HTTP() *httpClientCtx {
	hc := &httpClientCtx{
		c:    &s.httpC,
		uri:  fasthttp.AcquireURI(),
		args: fasthttp.AcquireArgs(),
		req:  fasthttp.AcquireRequest(),
		res:  fasthttp.AcquireResponse(),
	}

	if s.cfg.secure {
		hc.uri.SetScheme("https")
	} else {
		hc.uri.SetScheme("http")
	}

	hc.uri.SetHost(s.cfg.hostPort)

	return hc
}

type Callback func(ctx context.Context, statusCode int, body []byte)

type httpClientCtx struct {
	err  error
	c    *fasthttp.Client
	cb   Callback
	uri  *fasthttp.URI
	args *fasthttp.Args
	req  *fasthttp.Request
	res  *fasthttp.Response
}

func (hc *httpClientCtx) SetMethod(method string) *httpClientCtx {
	hc.req.Header.SetMethod(method)

	return hc
}

func (hc *httpClientCtx) SetPath(path string) *httpClientCtx {
	hc.uri.SetPath(path)

	return hc
}

func (hc *httpClientCtx) SetQuery(key, value string) *httpClientCtx {
	hc.args.Set(key, value)

	return hc
}

func (hc *httpClientCtx) SetHeader(key, value string) *httpClientCtx {
	hc.req.Header.Set(key, value)

	return hc
}

func (hc *httpClientCtx) SetBody(body []byte) *httpClientCtx {
	hc.req.SetBody(body)

	return hc
}

func (hc *httpClientCtx) Run(ctx context.Context) *httpClientCtx {
	// prepare the request
	hc.uri.SetQueryString(hc.args.String())
	hc.req.SetURI(hc.uri)

	// execute the request
	hc.err = hc.c.Do(hc.req, hc.res)

	// run the callback if is set
	if hc.err == nil && hc.cb != nil {
		hc.cb(ctx, hc.res.StatusCode(), hc.res.Body())
	}

	return hc
}

func (hc *httpClientCtx) Err() error {
	return hc.err
}

func (hc *httpClientCtx) StatusCode() int { return hc.res.StatusCode() }

func (hc *httpClientCtx) GetHeader(key string) string {
	return string(hc.res.Header.Peek(key))
}

// GetBody returns the body, but please note that the returned slice is only valid until
// Release is called. If you need to use the body after releasing httpClientCtx then
// use CopyBody method.
func (hc *httpClientCtx) GetBody() []byte {
	if hc.err != nil {
		return nil
	}

	return hc.res.Body()
}

func (hc *httpClientCtx) CopyBody(dst []byte) []byte {
	if hc.err != nil {
		return nil
	}

	dst = append(dst[:0], hc.res.Body()...)

	return dst
}

func (hc *httpClientCtx) Release() {
	fasthttp.ReleaseArgs(hc.args)
	fasthttp.ReleaseURI(hc.uri)
	fasthttp.ReleaseRequest(hc.req)
	fasthttp.ReleaseResponse(hc.res)
}

func (hc *httpClientCtx) SetCallback(cb Callback) *httpClientCtx {
	hc.cb = cb

	return hc
}

func (hc *httpClientCtx) DumpResponse() string {
	if hc.err != nil {
		return hc.err.Error()
	}

	return hc.res.String()
}

func (hc *httpClientCtx) DumpRequest() string {
	if hc.err != nil {
		return hc.err.Error()
	}

	return hc.req.String()
}

func (hc *httpClientCtx) AutoRun(s ronykit.RESTRouteSelector, m ronykit.Message) {}
