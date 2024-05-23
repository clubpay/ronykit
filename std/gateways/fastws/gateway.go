package fastws

import (
	"bytes"
	builtinErr "errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/buf"
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

func newGateway(b *bundle) *gateway {
	gw := &gateway{
		b:     b,
		conns: map[uint64]*wsConn{},
	}

	return gw
}

func (gw *gateway) reactFunc(wsc kit.Conn, payload *buf.Bytes, n int) {
	gw.b.d.OnMessage(wsc, (*payload.Bytes())[:n])
	payload.Release()
}

func (gw *gateway) getConnWrap(conn gnet.Conn) *wsConn {
	connID, ok := conn.Context().(uint64)
	if !ok {
		return nil
	}
	gw.Lock()
	cw := gw.conns[connID]
	gw.Unlock()

	return cw
}

func (gw *gateway) OnBoot(_ gnet.Engine) (action gnet.Action) {
	return gnet.None
}

func (gw *gateway) OnShutdown(_ gnet.Engine) {}

func (gw *gateway) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	wsc := newWebsocketConn(atomic.AddUint64(&gw.nextID, 1), c, gw.b.rpcOutFactory)
	c.SetContext(wsc.id)

	gw.Lock()
	gw.conns[wsc.id] = wsc
	gw.Unlock()

	gw.b.d.OnOpen(wsc)

	return nil, gnet.None
}

func (gw *gateway) OnClose(c gnet.Conn, _ error) (action gnet.Action) {
	connID, ok := c.Context().(uint64)
	if ok {
		gw.b.d.OnClose(connID)

		gw.Lock()
		delete(gw.conns, connID)
		gw.Unlock()
	}

	_ = c.Close()

	return gnet.Close
}

func (gw *gateway) OnTraffic(c gnet.Conn) gnet.Action {
	wsc := gw.getConnWrap(c)
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

	hdr, err = wsc.r.NextFrame()
	if err != nil {
		if builtinErr.Is(err, io.EOF) {
			return gnet.None
		}

		return gnet.Close
	}

	var p []byte
	if hdr.Fin {
		// No more frames will be read. Use fixed sized buffer to read payload.
		p = make([]byte, hdr.Length)
		// It is not possible to receive io.EOF here because Reader does not
		// return EOF if frame payload was successfully fetched.
		_, err = io.ReadFull(wsc.r, p)
	} else {
		// Frame is fragmented, thus use io.ReadAll behavior.
		var buff bytes.Buffer
		_, err = buff.ReadFrom(wsc.r)
		p = buff.Bytes()
	}
	if err != nil {
		return gnet.Close
	}

	wsc.msgs = append(wsc.msgs, wsutil.Message{OpCode: hdr.OpCode, Payload: p})

	for _, msg := range wsc.msgs {
		if msg.OpCode&(ws.OpText|ws.OpBinary) == 0 {
			continue
		}

		payloadBuffer := buf.GetLen(len(msg.Payload))
		payloadBuffer.CopyFrom(msg.Payload)

		go gw.reactFunc(wsc, payloadBuffer, len(msg.Payload))
	}

	wsc.msgs = wsc.msgs[:0]

	return gnet.None
}

func (gw *gateway) OnTick() (delay time.Duration, action gnet.Action) {
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
		if bytes.Equal(key, utils.S2B(headerOrigin)) {
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
