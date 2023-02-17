package stub

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/reflector"
	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

type RESTResponseHandler func(ctx context.Context, r RESTResponse) *Error

type RESTResponse interface {
	StatusCode() int
	GetBody() []byte
	GetHeader(key string) string
}

type RESTPreflightHandler func(r *fasthttp.Request)

type RESTCtx struct {
	cfg            restConfig
	err            *Error
	handlers       map[int]RESTResponseHandler
	defaultHandler RESTResponseHandler
	r              *reflector.Reflector
	dumpReq        io.Writer
	dumpRes        io.Writer

	// fasthttp entities
	c    *fasthttp.Client
	uri  *fasthttp.URI
	args *fasthttp.Args
	req  *fasthttp.Request
	res  *fasthttp.Response
}

func (hc *RESTCtx) SetMethod(method string) *RESTCtx {
	hc.req.Header.SetMethod(method)

	return hc
}

func (hc *RESTCtx) SetPath(path string) *RESTCtx {
	hc.uri.SetPath(path)

	return hc
}

func (hc *RESTCtx) GET(path string) *RESTCtx {
	hc.SetMethod(http.MethodGet)
	hc.SetPath(path)

	return hc
}

func (hc *RESTCtx) POST(path string) *RESTCtx {
	hc.SetMethod(http.MethodPost)
	hc.SetPath(path)

	return hc
}

func (hc *RESTCtx) PUT(path string) *RESTCtx {
	hc.SetMethod(http.MethodPut)
	hc.SetPath(path)

	return hc
}

func (hc *RESTCtx) PATCH(path string) *RESTCtx {
	hc.SetMethod(http.MethodPatch)
	hc.SetPath(path)

	return hc
}

func (hc *RESTCtx) OPTIONS(path string) *RESTCtx {
	hc.SetMethod(http.MethodOptions)
	hc.SetPath(path)

	return hc
}

func (hc *RESTCtx) SetQuery(key, value string) *RESTCtx {
	hc.args.Set(key, value)

	return hc
}

func (hc *RESTCtx) SetHeader(key, value string) *RESTCtx {
	hc.req.Header.Set(key, value)

	return hc
}

func (hc *RESTCtx) SetBody(body []byte) *RESTCtx {
	hc.req.SetBody(body)

	return hc
}

func (hc *RESTCtx) Run(ctx context.Context) *RESTCtx {
	// prepare the request
	hc.uri.SetQueryString(hc.args.String())
	hc.req.SetURI(hc.uri)

	if tp := hc.cfg.tp; tp != nil {
		tp.Inject(ctx, restTraceCarrier{r: &hc.req.Header})
	}

	// run preflights
	for _, pre := range hc.cfg.preflights {
		pre(hc.req)
	}

	// execute the request
	hc.err = WrapError(hc.c.Do(hc.req, hc.res))

	if hc.dumpReq != nil {
		_, _ = hc.req.WriteTo(hc.dumpReq)
	}
	if hc.dumpRes != nil {
		_, _ = hc.res.WriteTo(hc.dumpRes)
	}

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

// Err returns the error if any occurred during the execution.
func (hc *RESTCtx) Err() *Error {
	if hc.err == nil {
		return nil
	}

	return hc.err
}

// Error returns the error if any occurred during the execution.
func (hc *RESTCtx) Error() error {
	if hc.err == nil {
		return nil
	}

	return hc.err
}

// StatusCode returns the status code of the response
func (hc *RESTCtx) StatusCode() int { return hc.res.StatusCode() }

// GetHeader returns the header value for key in the response
func (hc *RESTCtx) GetHeader(key string) string {
	return string(hc.res.Header.Peek(key))
}

// GetBody returns the body, but please note that the returned slice is only valid until
// Release is called. If you need to use the body after releasing RESTCtx then
// use CopyBody method.
func (hc *RESTCtx) GetBody() []byte {
	if hc.err != nil {
		return nil
	}

	return hc.res.Body()
}

// CopyBody copies the body to `dst`. It creates a new slice and returns it if dst is nil.
func (hc *RESTCtx) CopyBody(dst []byte) []byte {
	if hc.err != nil {
		return nil
	}

	dst = append(dst[:0], hc.res.Body()...)

	return dst
}

// Release frees the allocates internal resources to be re-used.
// You MUST NOT refer to any method of this object after calling this method, if
// you call any method after Release has been called, the result is unpredictable.
func (hc *RESTCtx) Release() {
	fasthttp.ReleaseArgs(hc.args)
	fasthttp.ReleaseURI(hc.uri)
	fasthttp.ReleaseRequest(hc.req)
	fasthttp.ReleaseResponse(hc.res)
}

func (hc *RESTCtx) SetResponseHandler(statusCode int, h RESTResponseHandler) *RESTCtx {
	hc.handlers[statusCode] = h

	return hc
}

func (hc *RESTCtx) SetOKHandler(h RESTResponseHandler) *RESTCtx {
	hc.handlers[http.StatusOK] = h
	hc.handlers[http.StatusCreated] = h
	hc.handlers[http.StatusAccepted] = h

	return hc
}

func (hc *RESTCtx) DefaultResponseHandler(h RESTResponseHandler) *RESTCtx {
	hc.defaultHandler = h

	return hc
}

func (hc *RESTCtx) DumpResponse() string {
	return hc.res.String()
}

// DumpResponseTo accepts a writer and will write the response dump to it when Run is
// executed.
// Example:
//
//	httpCtx := s.REST().
//								DumpRequestTo(os.Stdout).
//								DumpResponseTo(os.Stdout).
//								GET("https//google.com").
//								Run(ctx)
//	defer httpCtx.Release()
//
// **YOU MUST NOT USE httpCtx after httpCtx.Release() is called.**
func (hc *RESTCtx) DumpResponseTo(w io.Writer) *RESTCtx {
	hc.dumpRes = w

	return hc
}

func (hc *RESTCtx) DumpRequest() string {
	if hc.err != nil {
		return hc.err.Error()
	}

	return hc.req.String()
}

// DumpRequestTo accepts a writer and will write the request dump to it when Run is
// executed.
//
// Please refer to DumpResponseTo
func (hc *RESTCtx) DumpRequestTo(w io.Writer) *RESTCtx {
	hc.dumpReq = w

	return hc
}

// AutoRun is a helper method, which fills the request based on the input arguments.
// It checks the route which is a path pattern, and fills the dynamic url params based on
// the `m`'s `tag` keys.
// Example:
//
//	type Request struct {
//			ID int64 `json:"id"`
//			Name string `json:"name"`
//	}
//
// AutoRun(
//
//		context.Background(),
//	  "/something/:id/:name",
//	  kit.JSON,
//	  &Request{ID: 10, Name: "customName"},
//
// )
//
// Is equivalent to:
//
// SetPath("/something/10/customName").
// Run(context.Background())
func (hc *RESTCtx) AutoRun(
	ctx context.Context, route string, enc kit.Encoding, m kit.Message,
) *RESTCtx {
	switch enc.Tag() {
	case kit.JSON.Tag():
		hc.SetHeader("Content-Type", "application/json")
	case kit.Proto.Tag():
		hc.SetHeader("Content-Type", "application/protobuf")
	}

	ref := hc.r.Load(m, enc.Tag())
	fields, ok := ref.ByTag(enc.Tag())
	if !ok {
		fields = ref.Obj()
	}

	usedParams := map[string]struct{}{}
	path := fillParams(
		route,
		func(key string) string {
			usedParams[key] = struct{}{}

			v := fields.Get(m, key)
			if v == nil {
				return "_"
			}

			return fmt.Sprintf("%v", v)
		},
	)
	hc.SetPath(path)

	switch utils.B2S(hc.req.Header.Method()) {
	case http.MethodGet:
		fields.WalkFields(
			func(key string, f reflector.FieldInfo) {
				_, ok := usedParams[key]
				if ok {
					return
				}

				v := fields.Get(m, key)
				if v == nil {
					return
				}

				hc.SetQuery(key, fmt.Sprintf("%v", v))
			},
		)
	default:
		var reqBody []byte
		switch enc {
		case kit.JSON:
			reqBody, _ = json.MarshalNoEscape(m)
		default:
			reqBody, _ = kit.MarshalMessage(m)
		}
		hc.SetBody(reqBody)
	}

	return hc.Run(ctx)
}

type restTraceCarrier struct {
	r *fasthttp.RequestHeader
}

func (t restTraceCarrier) Get(key string) string {
	return string(t.r.Peek(key))
}

func (t restTraceCarrier) Set(key string, value string) {
	t.r.Set(key, value)
}
