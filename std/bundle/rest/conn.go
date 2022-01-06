package rest

import (
	"mime/multipart"

	"github.com/ronaksoft/ronykit/utils"
	"github.com/valyala/fasthttp"
)

type conn struct {
	utils.SpinLock

	ctx *fasthttp.RequestCtx
}

func (c *conn) reset() {
	c.ctx = nil
}

func (c *conn) Walk(f func(key string, val string) bool) {
	stopCall := false
	c.ctx.Request.Header.VisitAll(
		func(key, value []byte) {
			if !stopCall {
				if !f(utils.B2S(key), utils.B2S(value)) {
					stopCall = true
				}
			}
		},
	)
}

func (c *conn) Get(key string) string {
	return utils.B2S(c.ctx.Request.Header.Peek(key))
}

func (c *conn) Set(key string, val string) {
	c.ctx.Response.Header.Set(key, val)
}

func (c *conn) SetStatusCode(code int) {
	c.ctx.Response.SetStatusCode(code)
}

func (c *conn) ConnID() uint64 {
	return c.ctx.ConnID()
}

func (c *conn) ClientIP() string {
	return c.ctx.RemoteIP().To4().String()
}

func (c *conn) Write(data []byte) (int, error) {
	c.ctx.SetBody(data)

	return len(data), nil
}

func (c *conn) Stream() bool {
	return false
}

func (c *conn) GetRequestURI() string {
	return utils.B2S(c.ctx.Request.RequestURI())
}

func (c *conn) GetMethod() string {
	return utils.B2S(c.ctx.Method())
}

func (c *conn) GetPath() string {
	return utils.B2S(c.ctx.URI().Path())
}

func (c *conn) Form() (*multipart.Form, error) {
	return c.ctx.MultipartForm()
}
