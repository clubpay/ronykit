package fasthttp

import (
	"mime/multipart"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/std/gateways/fasthttp/internal/realip"
	"github.com/valyala/fasthttp"
)

var strLocation = []byte(fasthttp.HeaderLocation)

type httpConn struct {
	utils.SpinLock

	ctx *fasthttp.RequestCtx
	rd  *routeData
}

var _ kit.RESTConn = (*httpConn)(nil)

func (c *httpConn) Walk(f func(key string, val string) bool) {
	stopCall := false
	c.ctx.Request.Header.VisitAll(
		func(key, value []byte) {
			if stopCall {
				return
			}

			stopCall = !f(utils.B2S(key), utils.B2S(value))
		},
	)
}

func (c *httpConn) WalkQueryParams(f func(key string, val string) bool) {
	stopCall := false
	c.ctx.QueryArgs().VisitAll(
		func(key, value []byte) {
			if stopCall {
				return
			}

			stopCall = !f(utils.B2S(key), utils.B2S(value))
		},
	)
}

func (c *httpConn) Get(key string) string {
	return utils.B2S(c.ctx.Request.Header.Peek(key))
}

func (c *httpConn) Set(key string, val string) {
	c.ctx.Response.Header.Set(key, val)
}

func (c *httpConn) SetStatusCode(code int) {
	c.ctx.Response.SetStatusCode(code)
}

func (c *httpConn) ConnID() uint64 {
	return c.ctx.ConnID()
}

func (c *httpConn) ClientIP() string {
	return realip.FromRequest(c.ctx)
}

func (c *httpConn) Write(data []byte) (int, error) {
	c.ctx.Response.AppendBody(data)

	return len(data), nil
}

func (c *httpConn) Stream() bool {
	return false
}

func (c *httpConn) GetHost() string {
	return utils.B2S(c.ctx.Host())
}

func (c *httpConn) GetRequestURI() string {
	c.ctx.URI().RequestURI()

	return utils.B2S(c.ctx.Request.RequestURI())
}

func (c *httpConn) GetMethod() string {
	return utils.B2S(c.ctx.Method())
}

func (c *httpConn) GetPath() string {
	return utils.B2S(c.ctx.URI().Path())
}

func (c *httpConn) Form() (*multipart.Form, error) {
	return c.ctx.MultipartForm()
}

func (c *httpConn) Redirect(statusCode int, url string) {
	u := fasthttp.AcquireURI()
	_ = u.Parse(nil, utils.S2B(url))
	c.ctx.Response.Header.SetCanonical(strLocation, u.FullURI())
	c.ctx.Response.SetStatusCode(statusCode)
	fasthttp.ReleaseURI(u)
}
