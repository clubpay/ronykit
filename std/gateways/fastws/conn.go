package fastws

import (
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
)

type wsConn struct {
	utils.SpinLock

	id            uint64
	kv            map[string]string
	c             gnet.Conn
	r             *wsutil.Reader
	w             *wsutil.Writer
	handshakeDone bool
}

func newWebsocketConn(id uint64, c gnet.Conn) *wsConn {
	wsc := &wsConn{
		w:  wsutil.NewWriter(c, ws.StateServerSide, ws.OpText),
		r:  wsutil.NewReader(c, ws.StateServerSide),
		id: id,
		kv: map[string]string{},
		c:  c,
	}

	return wsc
}

func (c *wsConn) Close() {
	_ = c.c.Close()
}

func (c *wsConn) ConnID() uint64 {
	return c.id
}

func (c *wsConn) ClientIP() string {
	addr := c.c.RemoteAddr()
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
