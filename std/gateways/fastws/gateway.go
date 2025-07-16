package fastws

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/gobwas/ws"
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
	wsc := newWebsocketConn(
		atomic.AddUint64(&gw.nextID, 1),
		c,
		gw.b.rpcOutFactory,
		gw.b.writeMode,
	)
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
		gw.b.l.Debugf("did not find ws conn for connID(%d)", utils.TryCast[uint64](c.Context()))

		return gnet.Close
	}

	if !wsc.isUpgraded() {
		err := wsc.upgrade(c)
		if err != nil {
			gw.b.l.Debugf("faild to upgrade websocket connID(%d): %v", utils.TryCast[uint64](c.Context()), err)

			return gnet.Close
		}

		return gnet.None
	}

	err := wsc.readBuffer(c)
	if err != nil {
		gw.b.l.Debugf("faild to read buffer websocket connID(%d): %v", utils.TryCast[uint64](c.Context()), err)

		return gnet.Close
	}

	err = wsc.executeMessages(c, gw.b.d)
	if err != nil {
		gw.b.l.Debugf("failed to execute message connID(%d): %v", utils.TryCast[uint64](c.Context()), err)

		return gnet.Close
	}

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
