package fasthttp

import (
	"bufio"
	"sync"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils/buf"
)

type sseHTTPConn struct {
	httpConn

	w     *bufio.Writer
	done  chan struct{}
	close sync.Once
}

var (
	_ kit.RESTConn = (*sseHTTPConn)(nil)
	_ kit.Conn     = (*sseHTTPConn)(nil)
)

func (c *sseHTTPConn) attachWriter(w *bufio.Writer) {
	c.w = w
}

func (c *sseHTTPConn) Stream() bool {
	return true
}

func (c *sseHTTPConn) Write(data []byte) (int, error) {
	if c.w == nil {
		return 0, kit.ErrWriteToClosedConn
	}

	err := writeSSEEvent(c.w, "", data)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

func (c *sseHTTPConn) WriteEnvelope(e *kit.Envelope) error {
	dataBuf := buf.GetCap(e.SizeHint())

	err := kit.EncodeMessage(e.GetMsg(), dataBuf)
	if err != nil {
		dataBuf.Release()

		return err
	}

	if c.w == nil {
		dataBuf.Release()

		return kit.ErrWriteToClosedConn
	}

	err = writeSSEEvent(c.w, sseEventMessage, *dataBuf.Bytes())
	dataBuf.Release()

	if err != nil {
		return err
	}

	e.WalkHdr(
		func(key string, val string) bool {
			c.ctx.Response.Header.Set(key, val)

			return true
		},
	)

	return nil
}

func (c *sseHTTPConn) signalDone() {
	c.close.Do(func() {
		close(c.done)
	})
}
