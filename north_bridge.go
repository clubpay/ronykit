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
	eh      ErrHandler
}

func (n *northBridge) OnOpen(c Conn) {
	// TODO: do we need to any thing
}

func (n *northBridge) OnClose(connID uint64) {
	// TODO:: do we need to do anything ?
}

func (n *northBridge) OnMessage(c Conn, msg []byte) error {
	if ce := n.l.Check(log.DebugLevel, "received message"); ce != nil {
		ce.Write(
			zap.Uint64("ConnID", c.ConnID()),
		)
	}

	dispatchFunc, err := n.d.Dispatch(c, msg)
	if err != nil {
		return err
	}

	ctx := acquireCtx(c)
	err = dispatchFunc(
		ctx,
		func(m Message, writeFunc WriteFunc, handlers ...Handler) {
			ctx.in = m
			ctx.wf = writeFunc
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
