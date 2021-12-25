package ronykit

import (
	"sync"

	"github.com/ronaksoft/ronykit/log"
	"go.uber.org/zap"
)

type northBridge struct {
	ctxPool sync.Pool
	l       log.Logger
	d       Dispatcher
	gw      Gateway
}

func (n *northBridge) OnOpen(c Conn) {
	// TODO: do we need to any thing
}

func (n *northBridge) OnClose(connID uint64) {
	// TODO:: do we need to do anything ?
}

func (n *northBridge) OnMessage(c Conn, streamID int64, msg []byte) error {
	if ce := n.l.Check(log.DebugLevel, "received message"); ce != nil {
		ce.Write(
			zap.Uint64("ConnID", c.ConnID()),
		)
	}

	dispatchFunc := n.d.Dispatch(c, streamID, msg)
	if dispatchFunc == nil {
		return nil
	}

	ctx := acquireCtx(c, streamID)
	err := dispatchFunc(
		ctx,
		func(m Message, flusher FlushFunc, handlers ...Handler) {
			ctx.in = m
			ctx.flusher = flusher
		Loop:
			for idx := range handlers {
				h := handlers[idx]
				for h != nil {
					h = h(ctx)
					if ctx.stopped {
						break Loop
					}
				}
			}

			return
		},
	)
	releaseCtx(ctx)

	return err
}
