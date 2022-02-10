package fastws

import (
	"bytes"
	"io"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/ants/v2"
	"github.com/panjf2000/gnet"
	"github.com/ronaksoft/ronykit/pools"
	"github.com/ronaksoft/ronykit/utils"
)

type gateway struct {
	utils.SpinLock

	b      *bundle
	nextID uint64
	conns  map[uint64]*wsConn
	goPool *ants.PoolWithFunc
}

func newGateway(b *bundle) (*gateway, error) {
	gw := &gateway{
		b:     b,
		conns: map[uint64]*wsConn{},
	}

	goPool, err := ants.NewPoolWithFunc(
		runtime.GOMAXPROCS(0)*1000,
		func(v interface{}) {
			gw.reactFunc(v.(*wsConn))
		},
		ants.WithExpiryDuration(time.Minute),
	)
	if err != nil {
		return nil, err
	}

	gw.goPool = goPool

	return gw, nil
}

func (e *gateway) reactFunc(wsc *wsConn) {
	if !wsc.c.handshakeDone {
		sp := acquireSwitchProtocol()
		_, err := sp.Upgrade(wsc.c)
		if err != nil {
			_ = wsc.c.Close()

			return
		}
		releaseSwitchProtocol(sp)

		wsc.c.handshakeDone = true

		return
	}

	hdr, err := wsc.r.NextFrame()
	if err != nil {
		_ = wsc.c.Close()

		return
	}

	payload := pools.Bytes.GetLen(int(hdr.Length))
	n, err := wsc.r.Read(payload)
	if err != nil && err != io.EOF {
		// FixME:: close connection ?!
		return
	}

	switch hdr.OpCode {
	case ws.OpClose:
		_ = wsc.c.Close()
	case ws.OpBinary, ws.OpText:
		e.b.d.OnMessage(wsc, payload[:n])
		pools.Bytes.Put(payload)
	}
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

func (e *gateway) OnInitComplete(server gnet.Server) (action gnet.Action) {
	return gnet.None
}

func (e *gateway) OnShutdown(server gnet.Server) {}

func (e *gateway) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	wsc := &wsConn{
		kv: map[string]string{},
		c: &wrapConn{
			handshakeDone: false,
			buf:           &bytes.Buffer{},
		},
	}
	wsc.w = wsutil.NewWriter(wsc.c, ws.StateServerSide, ws.OpText)
	wsc.r = wsutil.NewReader(wsc.c, ws.StateServerSide)
	wsc.id = atomic.AddUint64(&e.nextID, 1)
	wsc.c.c = c

	c.SetContext(wsc.id)

	e.Lock()
	e.conns[wsc.id] = wsc
	e.Unlock()

	e.b.d.OnOpen(wsc)

	return nil, gnet.None
}

func (e *gateway) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	connID, ok := c.Context().(uint64)
	if ok {
		e.b.d.OnClose(connID)

		e.Lock()
		delete(e.conns, connID)
		e.Unlock()
	}

	_ = c.Close()

	return gnet.Close
}

func (e *gateway) PreWrite(c gnet.Conn) {
	return
}

func (e *gateway) AfterWrite(c gnet.Conn, b []byte) {
	return
}

func (e *gateway) React(packet []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	wsc := e.getConnWrap(c)
	if wsc == nil {
		return nil, gnet.Close
	}

	wsc.c.buf.Write(packet)

	_ = e.goPool.Invoke(wsc)
	//go e.reactFunc(wsc)

	return nil, gnet.None
}

func (e *gateway) Tick() (delay time.Duration, action gnet.Action) {
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
