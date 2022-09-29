package stub

import (
	"context"
	"io"

	"github.com/clubpay/ronykit/utils/reflector"
	"github.com/valyala/fasthttp"
)

type WSResponseHandler func(ctx context.Context, r WSEnvelope) *Error

type WSEnvelope interface {
	GetPredicate() string
	GetBody() []byte
	GetHeader(key string) string
}

type WebsocketCtx struct {
	err            *Error
	handlers       map[int]WSResponseHandler
	defaultHandler WSResponseHandler
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
