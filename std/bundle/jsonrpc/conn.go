package jsonrpc

import (
	"github.com/panjf2000/gnet"
	"github.com/ronaksoft/ronykit/utils"
)

type connWrap struct {
	utils.SpinLock
	id uint64
	kv map[string]string
	c  gnet.Conn
}

func (c *connWrap) reset() {
	for k := range c.kv {
		delete(c.kv, k)
	}
	_ = c.c.Close()
}

func (c *connWrap) ConnID() uint64 {
	return c.id
}

func (c *connWrap) ClientIP() string {
	return c.c.RemoteAddr().String()
}

func (c *connWrap) Write(data []byte) (int, error) {
	err := c.c.AsyncWrite(data)

	return len(data), err
}

func (c *connWrap) Stream() bool {
	return true
}

func (c *connWrap) Walk(f func(key string, val string) bool) {
	c.Lock()
	defer c.Unlock()

	for k, v := range c.kv {
		if !f(k, v) {
			return
		}
	}
}

func (c *connWrap) Get(key string) string {
	c.Lock()
	v := c.kv[key]
	c.Unlock()

	return v
}

func (c *connWrap) Set(key string, val string) {
	c.Lock()
	c.kv[key] = val
	c.Unlock()
}
