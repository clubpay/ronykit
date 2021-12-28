package jsonrpc

import (
	"bytes"

	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet"
	"github.com/ronaksoft/ronykit/utils"
)

type wsConn struct {
	utils.SpinLock
	id uint64

	kv map[string]string
	c  *wrapConn
	r  *wsutil.Reader
	w  *wsutil.Writer
}

func (c *wsConn) reset() {
	for k := range c.kv {
		delete(c.kv, k)
	}
	c.c.reset()
}

func (c *wsConn) ConnID() uint64 {
	return c.id
}

func (c *wsConn) ClientIP() string {
	return c.c.c.RemoteAddr().String()
}

func (c *wsConn) Write(data []byte) (int, error) {
	n, err := c.w.Write(data)
	if err != nil {
		return n, err
	}

	err = c.w.Flush()

	return n, err
}

func (c *wsConn) Stream() bool {
	return true
}

func (c *wsConn) Walk(f func(key string, val string) bool) {
	c.Lock()
	defer c.Unlock()

	for k, v := range c.kv {
		if !f(k, v) {
			return
		}
	}
}

func (c *wsConn) Get(key string) string {
	c.Lock()
	v := c.kv[key]
	c.Unlock()

	return v
}

func (c *wsConn) Set(key string, val string) {
	c.Lock()
	c.kv[key] = val
	c.Unlock()
}

type wrapConn struct {
	c             gnet.Conn
	handshakeDone bool
	buf           *bytes.Buffer
}

func (c *wrapConn) Read(data []byte) (n int, err error) {
	rem := len(data)
	if rem == 0 {
		return 0, nil
	}

	bn, _ := c.buf.Read(data)
	if bn >= rem {
		return bn, nil
	}

	rem -= bn

	n, buf := c.c.ReadN(rem)
	copy(data[bn:], buf)

	return bn + n, nil
}

func (c *wrapConn) Write(data []byte) (n int, err error) {
	err = c.c.AsyncWrite(data)

	return len(data), err
}

func (c *wrapConn) Close() error {
	return c.c.Close()
}

func (c *wrapConn) reset() {
	c.handshakeDone = false
	c.buf.Reset()
	_ = c.c.Close()
	c.c = nil
}
