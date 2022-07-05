package stub

import (
	"context"

	"github.com/clubpay/ronykit"
	"github.com/valyala/fasthttp"
)

type RESTResponseHandler func(ctx context.Context, r RESTResponse) error

type RESTResponse interface {
	StatusCode() int
	GetBody() []byte
	GetHeader(key string) string
}

type restClientCtx struct {
	err            error
	c              *fasthttp.Client
	handlers       map[int]RESTResponseHandler
	defaultHandler RESTResponseHandler
	uri            *fasthttp.URI
	args           *fasthttp.Args
	req            *fasthttp.Request
	res            *fasthttp.Response
}

func (hc *restClientCtx) SetMethod(method string) *restClientCtx {
	hc.req.Header.SetMethod(method)

	return hc
}

func (hc *restClientCtx) SetPath(path string) *restClientCtx {
	hc.uri.SetPath(path)

	return hc
}

func (hc *restClientCtx) SetQuery(key, value string) *restClientCtx {
	hc.args.Set(key, value)

	return hc
}

func (hc *restClientCtx) SetHeader(key, value string) *restClientCtx {
	hc.req.Header.Set(key, value)

	return hc
}

func (hc *restClientCtx) SetBody(body []byte) *restClientCtx {
	hc.req.SetBody(body)

	return hc
}

func (hc *restClientCtx) Run(ctx context.Context) *restClientCtx {
	// prepare the request
	hc.uri.SetQueryString(hc.args.String())
	hc.req.SetURI(hc.uri)

	// execute the request
	hc.err = hc.c.Do(hc.req, hc.res)

	// run the response handler if is set
	statusCode := hc.res.StatusCode()
	if hc.err == nil {
		if h, ok := hc.handlers[statusCode]; ok {
			hc.err = h(ctx, hc)
		} else if hc.defaultHandler != nil {
			hc.err = hc.defaultHandler(ctx, hc)
		}
	}

	return hc
}

func (hc *restClientCtx) Err() error {
	return hc.err
}

// StatusCode returns the status code of the response
func (hc *restClientCtx) StatusCode() int { return hc.res.StatusCode() }

// GetHeader returns the header value for key in the response
func (hc *restClientCtx) GetHeader(key string) string {
	return string(hc.res.Header.Peek(key))
}

// GetBody returns the body, but please note that the returned slice is only valid until
// Release is called. If you need to use the body after releasing restClientCtx then
// use CopyBody method.
func (hc *restClientCtx) GetBody() []byte {
	if hc.err != nil {
		return nil
	}

	return hc.res.Body()
}

func (hc *restClientCtx) CopyBody(dst []byte) []byte {
	if hc.err != nil {
		return nil
	}

	dst = append(dst[:0], hc.res.Body()...)

	return dst
}

func (hc *restClientCtx) Release() {
	fasthttp.ReleaseArgs(hc.args)
	fasthttp.ReleaseURI(hc.uri)
	fasthttp.ReleaseRequest(hc.req)
	fasthttp.ReleaseResponse(hc.res)
}

func (hc *restClientCtx) SetResponseHandler(statusCode int, h RESTResponseHandler) *restClientCtx {
	hc.handlers[statusCode] = h

	return hc
}

func (hc *restClientCtx) DefaultResponseHandler(h RESTResponseHandler) *restClientCtx {
	hc.defaultHandler = h

	return hc
}

func (hc *restClientCtx) DumpResponse() string {
	if hc.err != nil {
		return hc.err.Error()
	}

	return hc.res.String()
}

func (hc *restClientCtx) DumpRequest() string {
	if hc.err != nil {
		return hc.err.Error()
	}

	return hc.req.String()
}

// AutoRun is a helper method, which fills the request based on the input arguments.
// It checks the route which is a path pattern, and fills the dynamic url params based on
// the `m`'s `tag` keys.
// Example:
// type Request struct {
//		ID int64 `json:"id"`
//		Name string `json:"name"`
// }
// AutoRun(
//	  "/something/:id/:name",
//	  "json",
//	  &Request{ID: 10, Name: "customName"},
// )
func (hc *restClientCtx) AutoRun(route string, m ronykit.Message) {}
