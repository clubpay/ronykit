package fasthttp

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/fasthttp/websocket"
)

type wsConn struct {
	utils.SpinLock
	kv            map[string]string
	id            uint64
	clientIP      string
	c             *websocket.Conn
	rpcOutFactory kit.OutgoingRPCFactory
}

var _ kit.Conn = (*wsConn)(nil)

func (w *wsConn) Close() {
	w.Lock()
	w.c = nil
	w.Unlock()
}

func (w *wsConn) ConnID() uint64 {
	return w.id
}

func (w *wsConn) ClientIP() string {
	return w.clientIP
}

func (w *wsConn) Write(data []byte) (int, error) {
	var err error

	w.Lock()
	if w.c != nil {
		err = w.c.WriteMessage(websocket.TextMessage, data)
	} else {
		err = kit.ErrWriteToClosedConn
	}
	w.Unlock()
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

func (w *wsConn) WriteEnvelope(e *kit.Envelope) error {
	outC := w.rpcOutFactory()
	outC.InjectMessage(e.GetMsg())
	outC.SetID(e.GetID())
	e.WalkHdr(
		func(key string, val string) bool {
			outC.SetHdr(key, val)

			return true
		},
	)

	data, err := outC.Marshal()
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	outC.Release()

	return err
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
