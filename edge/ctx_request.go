package edge

import (
	"sync"

	"github.com/ronaksoft/ronykit"
)

type RequestCtx struct {
	sync.Mutex
	streamID int64
	conn     ronykit.Conn
	nb       *northBridge

	outBuf []ronykit.Envelope

	stopped bool
}

// Flush writes all the buffered envelopes to the wire
func (ctx *RequestCtx) Flush() (err error) {
	ctx.Lock()
	if len(ctx.outBuf) > 0 {
		err = ctx.nb.d.Serialize(ctx.conn, ctx.streamID, ctx.outBuf...)

		// if there is an envelope pool release the envelope into the pool
		if ctx.nb.ep != nil {
			for _, e := range ctx.outBuf {
				ctx.nb.ep.Release(e)
			}
		}

		// reset the buffer
		ctx.outBuf = ctx.outBuf[:0]
	}
	ctx.Unlock()

	return
}

// Push pushes the envelope into the RequestCtx buffer, which will be flushed to the connection
// at the end of the RequestCtx lifecycle, or by explicitly calling Flush func.
func (ctx *RequestCtx) Push(e ronykit.Envelope) {
	ctx.Lock()
	ctx.outBuf = append(ctx.outBuf, e)
	ctx.Unlock()
}

// Write writes the envelope to the wire after encoding it
func (ctx *RequestCtx) Write(e ronykit.Envelope) error {
	return ctx.nb.d.Serialize(ctx.conn, ctx.streamID, e)
}

// StopExecution stops the execution of the next handlers.
func (ctx *RequestCtx) StopExecution() {
	ctx.stopped = true
}
