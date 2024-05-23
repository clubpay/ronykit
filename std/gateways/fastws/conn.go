package fastws

import (
	"io"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/errors"
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
	rpcOutFactory kit.OutgoingRPCFactory
	msgs          []wsutil.Message
}

var _ kit.Conn = (*wsConn)(nil)

func newWebsocketConn(
	id uint64, c gnet.Conn,
	rpcOutFactory kit.OutgoingRPCFactory,
) *wsConn {
	wsc := &wsConn{
		w:             wsutil.NewWriter(c, ws.StateServerSide, ws.OpText),
		id:            id,
		kv:            map[string]string{},
		c:             c,
		rpcOutFactory: rpcOutFactory,
	}

	wsc.r = &wsutil.Reader{
		Source:    c,
		State:     ws.StateServerSide,
		CheckUTF8: true,
		OnIntermediate: func(hdr ws.Header, src io.Reader) error {
			if hdr.OpCode.IsControl() {
				return wsutil.ControlHandler{
					Src:                 wsc.r,
					Dst:                 wsc.c,
					State:               wsc.r.State,
					DisableSrcCiphering: true,
				}.Handle(hdr)
			}

			bts, err := io.ReadAll(src)
			if err != nil {
				return err
			}
			wsc.msgs = append(wsc.msgs, wsutil.Message{OpCode: hdr.OpCode, Payload: bts})

			return nil
		},
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

func (c *wsConn) WriteEnvelope(e *kit.Envelope) error {
	outC := c.rpcOutFactory()
	outC.InjectMessage(e.GetMsg())
	outC.SetID(e.GetID())
	e.WalkHdr(func(key string, val string) bool {
		outC.SetHdr(key, val)

		return true
	})

	data, err := outC.Marshal()
	if err != nil {
		return errors.Wrap(kit.ErrEncodeOutgoingMessageFailed, err)
	}

	_, err = c.Write(data)
	outC.Release()

	return err
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
