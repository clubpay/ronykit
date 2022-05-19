package fasthttp

import (
	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/utils"
	"github.com/fasthttp/websocket"
)

type wsConn struct {
	utils.SpinLock
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
	defer recoverPanic()

	w.Lock()
	err := w.c.WriteMessage(websocket.TextMessage, data)
	w.Unlock()
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

func (w *wsConn) Stream() bool {
	return true
}

func (w *wsConn) Walk(f func(key string, val string) bool) {
	w.Lock()
	for k, v := range w.kv {
		if !f(k, v) {
			break
		}
	}
	w.Unlock()
}

func (w *wsConn) Get(key string) string {
	w.Lock()
	v := w.kv[key]
	w.Unlock()

	return v
}

func (w *wsConn) Set(key string, val string) {
	w.Lock()
	w.kv[key] = val
	w.Unlock()
}

func recoverPanic() {
	if r := recover(); r != nil {
		print(r)
	}
}
