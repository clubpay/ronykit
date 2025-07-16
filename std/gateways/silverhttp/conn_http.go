package silverhttp

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/buf"
	"github.com/clubpay/ronykit/std/gateways/silverhttp/realip"
	"github.com/go-www/silverlining"
)

type httpConn struct {
	utils.SpinLock

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

func (c *httpConn) WalkQueryParams(f func(key string, val string) bool) {
	stopCall := false
	for _, h := range c.ctx.QueryParams() {
		if !stopCall {
			if !f(utils.B2S(h.Key), utils.B2S(h.Value)) {
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
	return c.ctx.ConnID()
}

func (c *httpConn) ClientIP() string {
	return realip.FromRequest(c.ctx)
}

func (c *httpConn) Write(data []byte) (int, error) {
	c.ctx.SetContentLength(len(data))

	_, err := c.ctx.Write(data)

	return len(data), err
}

func (c *httpConn) WriteEnvelope(e *kit.Envelope) error {
	dataBuf := buf.GetCap(e.SizeHint())

	err := kit.EncodeMessage(e.GetMsg(), dataBuf)
	if err != nil {
		return err
	}

	resHdr := c.ctx.ResponseHeaders()

	e.WalkHdr(
		func(key string, val string) bool {
			resHdr.Set(key, val)

			return true
		},
	)

	c.ctx.SetContentLength(dataBuf.Len())
	_, err = c.ctx.Write(utils.PtrVal(dataBuf.Bytes()))

	return err
}

func (c *httpConn) Stream() bool {
	return false
}

func (c *httpConn) GetHost() string {
	return c.ctx.Host()
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

func (c *httpConn) Redirect(statusCode int, url string) {
	c.ctx.Redirect(statusCode, url)
}
