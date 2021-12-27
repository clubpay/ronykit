package rest

import (
	"mime/multipart"

	"github.com/ronaksoft/ronykit/utils"
	"github.com/valyala/fasthttp"
)

type conn struct {
	utils.SpinLock

	kv  map[string]string
	ctx *fasthttp.RequestCtx
}

func (c *conn) reset() {
	for k := range c.kv {
		delete(c.kv, k)
	}
	c.ctx = nil
}

func (c *conn) Walk(f func(key string, val string) bool) {
	c.Lock()
	defer c.Unlock()
	for k, v := range c.kv {
		if !f(k, v) {
			return
		}
	}
}

func (c *conn) Get(key string) string {
	c.Lock()
	v := c.kv[key]
	c.Unlock()

	return v
}

func (c *conn) Set(key string, val string) {
	c.Lock()
	c.kv[key] = val
	c.Unlock()
}

func (c *conn) ConnID() uint64 {
	return c.ctx.ConnID()
}

func (c *conn) ClientIP() string {
	return c.ctx.RemoteIP().To4().String()
}

func (c *conn) Write(_ int64, data []byte) error {
	c.ctx.SetBody(data)

	return nil
}

func (c *conn) Stream() bool {
	return false
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

func (c *conn) WriteHeader(key, val string) {
	c.ctx.Response.Header.Set(key, val)
}
