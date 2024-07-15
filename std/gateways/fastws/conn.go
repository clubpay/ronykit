package fastws

import (
	"bytes"
	"io"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/errors"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/buf"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/buffer/ring"
)

const ringBufInitialSize = 4 << 10

type wsConn struct {
	utils.SpinLock

	id            uint64
	kv            map[string]string
	rpcOutFactory kit.OutgoingRPCFactory
	clientIP      string

	// websocketCodec
	handshakeDone bool
	readBuff      *ring.Buffer
	msgBuff       *buf.Bytes
	currHead      *ws.Header
	w             *wsutil.Writer
}

var (
	_ kit.Conn    = (*wsConn)(nil)
	_ kit.RPCConn = (*wsConn)(nil)
)

func newWebsocketConn(
	id uint64, c gnet.Conn,
	rpcOutFactory kit.OutgoingRPCFactory,
	writeMode ws.OpCode,
) *wsConn {
	wsc := &wsConn{
		w:             wsutil.NewWriter(c, ws.StateServerSide, writeMode),
		id:            id,
		kv:            map[string]string{},
		readBuff:      ring.New(ringBufInitialSize),
		msgBuff:       buf.GetCap(ringBufInitialSize),
		rpcOutFactory: rpcOutFactory,
	}

	if addr := c.RemoteAddr(); addr != nil {
		wsc.clientIP = addr.String()
	}

	return wsc
}

func (wsc *wsConn) readBuffer(c gnet.Conn) error {
	buff, err := c.Next(c.InboundBuffered())
	if err != nil {
		return err
	}

	_, err = wsc.readBuff.Write(buff)

	return err
}

func (wsc *wsConn) isUpgraded() bool {
	return wsc.handshakeDone
}

func (wsc *wsConn) upgrade(c gnet.Conn) error {
	sp := acquireSwitchProtocol()
	if _, err := sp.Upgrade(c); err != nil {
		return err
	}
	releaseSwitchProtocol(sp)

	wsc.handshakeDone = true

	return nil
}

func (wsc *wsConn) nextHeader() error {
	if wsc.currHead != nil {
		return nil
	}

	if wsc.readBuff.Buffered() < ws.MinHeaderSize {
		return nil
	}

	// we can read the header for sure
	if wsc.readBuff.Buffered() >= ws.MaxHeaderSize {
		head, err := ws.ReadHeader(wsc.readBuff)
		if err != nil {
			return err
		}

		wsc.currHead = &head

		return nil
	}

	// we need to check if there is header in the buffer
	tmp := bytes.NewReader(wsc.readBuff.Bytes())
	preLen := tmp.Len()
	head, err := ws.ReadHeader(tmp)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil
		}

		return err
	}

	skipN := preLen - tmp.Len()
	_, err = wsc.readBuff.Discard(skipN)
	if err != nil {
		return err
	}

	wsc.currHead = &head

	return nil
}

func (wsc *wsConn) handleControlMessage(c gnet.Conn) error {
	buff := buf.GetLen(int(wsc.currHead.Length))
	defer buff.Release()

	if wsc.currHead.Length > 0 {
		_, err := wsc.readBuff.Read(*buff.Bytes())
		if err != nil {
			return err
		}
	}

	err := wsutil.HandleClientControlMessage(
		c,
		wsutil.Message{
			OpCode:  wsc.currHead.OpCode,
			Payload: *buff.Bytes(),
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (wsc *wsConn) executeMessages(c gnet.Conn, d kit.GatewayDelegate) error {
	for {
		err := wsc.nextHeader()
		if err != nil {
			return err
		}

		if wsc.currHead == nil {
			return nil
		}

		// if it is a control message then let's handle it
		if wsc.currHead.Fin && wsc.currHead.OpCode.IsControl() {
			err = wsc.handleControlMessage(c)
			if err != nil {
				return err
			}

			wsc.currHead = nil

			continue
		}

		dataLen := int(wsc.currHead.Length)
		if dataLen > 0 {
			if dataLen > wsc.readBuff.Buffered() {
				return nil
			}

			tmpBuff := buf.GetLen(8192)
			stPos := wsc.msgBuff.Len()
			written, err := io.CopyBuffer(wsc.msgBuff, io.LimitReader(wsc.readBuff, wsc.currHead.Length), *tmpBuff.Bytes())
			tmpBuff.Release()
			if err != nil {
				return err
			}
			if written < wsc.currHead.Length && err == nil {
				// src stopped early; must have been EOF.
				return io.EOF
			}

			endPos := wsc.msgBuff.Len()
			ws.Cipher(utils.PtrVal(wsc.msgBuff.Bytes())[stPos:endPos], wsc.currHead.Mask, 0)
		}

		if wsc.currHead.Fin {
			msgBuff := wsc.msgBuff
			wsc.msgBuff = buf.GetCap(wsc.msgBuff.Cap())
			go wsc.execMessage(d, msgBuff)
		}

		// reset the current head
		wsc.currHead = nil
	}
}

func (wsc *wsConn) execMessage(d kit.GatewayDelegate, msgBuff *buf.Bytes) {
	d.OnMessage(wsc, *msgBuff.Bytes())
	msgBuff.Release()
}

func (wsc *wsConn) ConnID() uint64 {
	return wsc.id
}

func (wsc *wsConn) ClientIP() string {
	return wsc.clientIP
}

func (wsc *wsConn) Write(data []byte) (int, error) {
	wsc.Lock()
	defer wsc.Unlock()

	n, err := wsc.w.Write(data)
	if err != nil {
		return n, err
	}

	err = wsc.w.Flush()

	return n, err
}

func (wsc *wsConn) WriteEnvelope(e *kit.Envelope) error {
	outC := wsc.rpcOutFactory()
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

	_, err = wsc.Write(data)
	outC.Release()

	return err
}

func (wsc *wsConn) Stream() bool {
	return true
}

func (wsc *wsConn) Walk(f func(key string, val string) bool) {
	wsc.Lock()
	defer wsc.Unlock()

	for k, v := range wsc.kv {
		if !f(k, v) {
			return
		}
	}
}

func (wsc *wsConn) Get(key string) string {
	wsc.Lock()
	v := wsc.kv[key]
	wsc.Unlock()

	return v
}

func (wsc *wsConn) Set(key string, val string) {
	wsc.Lock()
	wsc.kv[key] = val
	wsc.Unlock()
}
