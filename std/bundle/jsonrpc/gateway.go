package jsonrpc

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet"
	"github.com/ronaksoft/ronykit/utils"
)

type gateway struct {
	utils.SpinLock

	b        *bundle
	nextID   uint64
	conns    map[uint64]*connWrap
	connPool sync.Pool
}

func (e *gateway) getConnWrap(conn gnet.Conn) *connWrap {
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
	cw, ok := e.connPool.Get().(*connWrap)
	if !ok {
		cw = &connWrap{
			kv: map[string]string{},
		}
	}
	cw.id = atomic.AddUint64(&e.nextID, 1)
	cw.c = c

	//ws.Upgrader.Upgrade(c.SendTo())

	c.SetContext(cw.id)

	e.Lock()
	e.conns[cw.id] = cw
	e.Unlock()

	e.b.d.OnOpen(cw)

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

	return gnet.Close
}

func (e *gateway) PreWrite(c gnet.Conn) {
	return
}

func (e *gateway) AfterWrite(c gnet.Conn, b []byte) {
	return
}

func (e *gateway) React(packet []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	cw := e.getConnWrap(c)
	if cw == nil {
		return nil, gnet.Close
	}

	_ = e.b.d.OnMessage(cw, packet)

	return nil, gnet.None
}

func (e *gateway) Tick() (delay time.Duration, action gnet.Action) {
	return 0, gnet.None
}
