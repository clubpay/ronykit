package fastws

import (
	"bytes"

	"github.com/clubpay/ronykit/utils"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet"
)

type wsConn struct {
	utils.SpinLock

	id uint64
	kv map[string]string
	c  *wrapConn
	r  *wsutil.Reader
	w  *wsutil.Writer
}

func newWebsocketConn(id uint64, c gnet.Conn) *wsConn {
	wc := &wrapConn{
		buf: &bytes.Buffer{},
		c:   c,
	}
	wsc := &wsConn{
		w:  wsutil.NewWriter(wc, ws.StateServerSide, ws.OpText),
		r:  wsutil.NewReader(wc, ws.StateServerSide),
		id: id,
		kv: map[string]string{},
		c:  wc,
	}

	return wsc
}

func (c *wsConn) Close() {
	_ = c.c.c.Close()
}

func (c *wsConn) ConnID() uint64 {
	return c.id
}

func (c *wsConn) ClientIP() string {
	addr := c.c.c.RemoteAddr()
	if addr == nil {
		return ""
	}

	return addr.String()
}

func (c *wsConn) Write(data []byte) (int, error) {
	c.Lock()
	defer c.Unlock()

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
	handshakeDone bool
	c             gnet.Conn
	buf           *bytes.Buffer
}

func (c *wrapConn) Read(data []byte) (n int, err error) {
	n, _ = c.buf.Read(data)

	return
}

func (c *wrapConn) Write(data []byte) (n int, err error) {
	err = c.c.AsyncWrite(data)

	return len(data), err
}
