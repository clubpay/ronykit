package ronykit

import (
	"github.com/clubpay/ronykit/internal/errors"
)

// Gateway is main component of the EdgeServer. Without Gateway, the EdgeServer is not functional. You can use
// some standard bundles in std/bundle path. However, if you need special handling of communication
// between your server and the clients you are free to implement your own Gateway.
type Gateway interface {
	Bundle
	// Subscribe will be called by the EdgeServer. These delegate functions
	// must be called by the Gateway implementation. In other words, Gateway communicates
	// with EdgeServer through the GatewayDelegate methods.
	//
	// NOTE: This func will be called only once and before calling Start function.
	Subscribe(d GatewayDelegate)
	// Dispatch receives the messages from external clients and runs the execFunc with appropriate
	// arguments. The user of the Gateway does not need to implement this. If you are using some
	// standard bundles like std/gateway/fasthttp or std/gateway/fastws then all the implementation
	// is taken care of.
	Dispatch(ctx *Context, in []byte) (ExecuteArg, error)
}

// GatewayDelegate is the delegate that connects the Gateway to the rest of the system.
type GatewayDelegate interface {
	// OnOpen must be called whenever a new connection is established.
	OnOpen(c Conn)
	// OnClose must be called whenever the connection is gone.
	OnClose(connID uint64)
	// OnMessage must be called whenever a new message arrives.
	OnMessage(c Conn, wf WriteFunc, msg []byte)
}

// northBridge is a container component that connects EdgeServer with a Gateway type Bundle.
type northBridge struct {
	ctxPool
	l  Logger
	eh ErrHandler
	cr contractResolver
	gw Gateway
}

var _ GatewayDelegate = (*northBridge)(nil)

func (n *northBridge) OnOpen(c Conn) {
	// Maybe later we can do something
}

func (n *northBridge) OnClose(connID uint64) {
	// Maybe later we can do something
}

func (n *northBridge) OnMessage(conn Conn, wf WriteFunc, msg []byte) {
	ctx := n.acquireCtx(conn)
	ctx.wf = wf

	arg, err := n.gw.Dispatch(ctx, msg)
	if err != nil {
		n.eh(ctx, errors.Wrap(ErrDispatchFailed, err))
	} else {
		ctx.execute(arg, n.cr(arg.ContractID))
	}

	n.releaseCtx(ctx)
}

var (
	ErrNoHandler                   = errors.New("handler is not set for request")
	ErrWriteToClosedConn           = errors.New("write to closed connection")
	ErrDecodeIncomingMessageFailed = errors.New("decoding the incoming message failed")
	ErrEncodeOutgoingMessageFailed = errors.New("encoding the outgoing message failed")
	ErrDispatchFailed              = errors.New("dispatch failed")
)
