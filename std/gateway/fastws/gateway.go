package fastws

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/clubpay/ronykit/utils"
	"github.com/clubpay/ronykit/utils/buf"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
)

type gateway struct {
	utils.SpinLock

	b      *bundle
	nextID uint64
	conns  map[uint64]*wsConn
}

func newGateway(b *bundle) (*gateway, error) {
	gw := &gateway{
		b:     b,
		conns: map[uint64]*wsConn{},
	}

	return gw, nil
}

func (e *gateway) reactFunc(wsc *wsConn, payload *buf.Bytes, n int) {
	e.b.d.OnMessage(wsc, (*payload.Bytes())[:n])
	payload.Release()
}

func (e *gateway) getConnWrap(conn gnet.Conn) *wsConn {
	connID, ok := conn.Context().(uint64)
	if !ok {
		return nil
	}
	e.Lock()
	cw := e.conns[connID]
	e.Unlock()

	return cw
}

func (e *gateway) OnBoot(_ gnet.Engine) (action gnet.Action) {
	return gnet.None
}

func (e *gateway) OnShutdown(_ gnet.Engine) {}

func (e *gateway) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	wsc := newWebsocketConn(atomic.AddUint64(&e.nextID, 1), c)
	c.SetContext(wsc.id)

	e.Lock()
	e.conns[wsc.id] = wsc
	e.Unlock()

	e.b.d.OnOpen(wsc)

	return nil, gnet.None
}

func (e *gateway) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	connID, ok := c.Context().(uint64)
	if ok {
		e.b.d.OnClose(connID)

		e.Lock()
		delete(e.conns, connID)
		e.Unlock()
	}

	_ = c.Close(nil)

	return gnet.Close
}

func (e *gateway) OnTraffic(c gnet.Conn) gnet.Action {
	wsc := e.getConnWrap(c)
	if wsc == nil {
		return gnet.Close
	}

	if !wsc.handshakeDone {
		sp := acquireSwitchProtocol()
		_, err := sp.Upgrade(wsc.c)
		if err != nil {
			wsc.Close()

			return gnet.Close
		}
		releaseSwitchProtocol(sp)

		wsc.handshakeDone = true

		return gnet.None
	}

	var (
		err error
		hdr ws.Header
	)

	for {
		hdr, err = wsc.r.NextFrame()
		if err != nil {
			return gnet.Close
		}
		if hdr.OpCode.IsControl() {
			wsc.r.OnIntermediate = func(header ws.Header, reader io.Reader) error {
				return wsutil.ControlHandler{
					Src:                 wsc.r,
					Dst:                 wsc.c,
					State:               wsc.r.State,
					DisableSrcCiphering: true,
				}.Handle(header)
			}
			if err := wsc.r.OnIntermediate(hdr, wsc.r); err != nil {
				return gnet.Close
			}

			continue
		}
		if hdr.OpCode&(ws.OpText|ws.OpBinary) == 0 {
			if err := wsc.r.Discard(); err != nil {
				return gnet.Close
			}

			continue
		}

		break
	}

	payloadBuffer := buf.GetLen(int(hdr.Length))
	n, err := wsc.r.Read(*payloadBuffer.Bytes())
	if err != nil && err != io.EOF {
		return gnet.None
	}

	go e.reactFunc(wsc, payloadBuffer, n)

	return gnet.None
}

func (e *gateway) OnTick() (delay time.Duration, action gnet.Action) {
	return 0, gnet.None
}

const (
	headerOrigin                   = "Origin"
	headerAccessControlAllowOrigin = "Access-Control-Allow-Origin"
)

type SwitchProtocol struct {
	u   ws.Upgrader
	hdr http.Header
}

func newSwitchProtocol() *SwitchProtocol {
	sp := &SwitchProtocol{
		hdr: http.Header{},
		u:   ws.Upgrader{},
	}

	sp.u.OnHeader = func(key, value []byte) error {
		switch {
		case bytes.Equal(key, utils.S2B(headerOrigin)):
			sp.hdr.Set(headerAccessControlAllowOrigin, string(value))
		}

		return nil
	}
	sp.u.OnBeforeUpgrade = func() (header ws.HandshakeHeader, err error) {
		return ws.HandshakeHeaderHTTP(sp.hdr), nil
	}

	return sp
}

func (sp *SwitchProtocol) Upgrade(conn io.ReadWriter) (hs ws.Handshake, err error) {
	return sp.u.Upgrade(conn)
}

var switchProtocolPool = sync.Pool{}

func acquireSwitchProtocol() *SwitchProtocol {
	sp, ok := switchProtocolPool.Get().(*SwitchProtocol)
	if !ok {
		sp = newSwitchProtocol()
	}

	return sp
}

func releaseSwitchProtocol(sp *SwitchProtocol) {
	for k := range sp.hdr {
		delete(sp.hdr, k)
	}

	switchProtocolPool.Put(sp)
}
