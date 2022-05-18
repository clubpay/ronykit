package fasthttp

import (
	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/utils"
	"github.com/fasthttp/websocket"
)

type wsConn struct {
	kvm      utils.SpinLock
	kv       map[string]string
	id       uint64
	clientIP string
	c        *websocket.Conn
}

var _ ronykit.Conn = (*wsConn)(nil)

func (w *wsConn) ConnID() uint64 {
	return w.id
}

func (w *wsConn) ClientIP() string {
	return w.clientIP
}

func (w *wsConn) Write(data []byte) (int, error) {
	err := w.c.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

func (w *wsConn) Stream() bool {
	return true
}

func (w *wsConn) Walk(f func(key string, val string) bool) {
	w.kvm.Lock()
	for k, v := range w.kv {
		if !f(k, v) {
			break
		}
	}
	w.kvm.Unlock()
}

func (w *wsConn) Get(key string) string {
	w.kvm.Lock()
	v := w.kv[key]
	w.kvm.Unlock()

	return v
}

func (w *wsConn) Set(key string, val string) {
	w.kvm.Lock()
	w.kv[key] = val
	w.kvm.Unlock()
}
