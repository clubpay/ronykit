package silverhttp

import (
	"mime/multipart"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/std/gateway/silverhttp/realip"
	"github.com/go-www/silverlining"
)

type httpConn struct {
	utils.SpinLock
	id uint64

	ctx *silverlining.Context
}

var _ kit.RESTConn = (*httpConn)(nil)

func (c *httpConn) Walk(f func(key string, val string) bool) {
	stopCall := false
	for _, h := range c.ctx.RequestHeaders().List() {
		if !stopCall {
			if !f(utils.B2S(h.Name), utils.B2S(h.RawValue)) {
				stopCall = true
			}
		}
	}
}

func (c *httpConn) Get(key string) string {
	v, ok := c.ctx.RequestHeaders().GetBytes(utils.S2B(key))
	if ok {
		return utils.B2S(v)
	}

	return ""
}

func (c *httpConn) Set(key string, val string) {
	c.ctx.ResponseHeaders().Set(key, val)
}

func (c *httpConn) SetStatusCode(code int) {
	c.ctx.WriteHeader(code)
}

func (c *httpConn) ConnID() uint64 {
	return c.id
}

func (c *httpConn) ClientIP() string {

	return realip.FromRequest(c.ctx)
}

func (c *httpConn) Write(data []byte) (int, error) {
	c.ctx.SetContentLength(len(data))
	_, err := c.ctx.Write(data)

	return len(data), err
}

func (c *httpConn) Stream() bool {
	return false
}

func (c *httpConn) GetHost() string {
	return ""
}

func (c *httpConn) GetRequestURI() string {
	return utils.B2S(c.ctx.RawURI())
}

func (c *httpConn) GetMethod() string {
	return c.ctx.Method().String()
}

func (c *httpConn) GetPath() string {
	return utils.B2S(c.ctx.Path())
}

func (c *httpConn) Form() (*multipart.Form, error) {
	r, err := c.ctx.MultipartReader()
	if err != nil {
		return nil, err
	}

	return r.ReadForm(maxMimeFormSize)
}

func (c *httpConn) Redirect(statusCode int, url string) {
	c.ctx.Redirect(statusCode, url)
}
