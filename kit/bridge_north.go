package kit

import (
	"context"
	"sync"

	"github.com/clubpay/ronykit/kit/errors"
)

type WriteFunc func(conn Conn, e *Envelope) error

type GatewayStartConfig struct {
	ReusePort bool
}

// Gateway is main component of the EdgeServer. Without Gateway, the EdgeServer is not functional. You can use
// some standard bundles in std/bundle path. However, if you need special handling of communication
// between your server and the clients you are free to implement your own Gateway.
type Gateway interface {
	// Start starts the gateway to accept connections.
	Start(ctx context.Context, cfg GatewayStartConfig) error
	// Shutdown shuts down the gateway gracefully.
	Shutdown(ctx context.Context) error
	// Register registers the route in the Bundle. This is how Bundle get information
	// about the services and their contracts.
	Register(
		serviceName, contractID string, enc Encoding, sel RouteSelector, input Message,
	)
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
	ConnDelegate
	// OnMessage must be called whenever a new message arrives.
	OnMessage(c Conn, wf WriteFunc, msg []byte)
}

type ConnDelegate interface {
	// OnOpen must be called whenever a new connection is established.
	OnOpen(c Conn)
	// OnClose must be called whenever the connection is gone.
	OnClose(connID uint64)
}

// northBridge is a container component that connects EdgeServer with a Gateway type Bundle.
type northBridge struct {
	ctxPool
	wg *sync.WaitGroup
	eh ErrHandlerFunc
	c  map[string]Contract
	gw Gateway
	sb *southBridge
	cd ConnDelegate
}

var _ GatewayDelegate = (*northBridge)(nil)

func (n *northBridge) OnOpen(c Conn) {
	if n.cd == nil {
		return
	}

	n.cd.OnOpen(c)
}

func (n *northBridge) OnClose(connID uint64) {
	if n.cd == nil {
		return
	}

	n.cd.OnClose(connID)
}

func (n *northBridge) OnMessage(conn Conn, wf WriteFunc, msg []byte) {
	n.wg.Add(1)
	ctx := n.acquireCtx(conn)
	ctx.wf = wf
	ctx.sb = n.sb
	ctx.rawData = msg

	arg, err := n.gw.Dispatch(ctx, msg)
	switch err {
	case nil:
		ctx.execute(arg, n.c[arg.ContractID])
	case ErrPreflight:
		// If this is a Preflight request we ignore executing it. This is a workaround
		// for CORS Preflight requests.
	default:
		n.eh(ctx, errors.Wrap(ErrDispatchFailed, err))
	}

	n.releaseCtx(ctx)
	n.wg.Done()
}

var (
	ErrNoHandler                     = errors.New("handler is not set for request")
	ErrWriteToClosedConn             = errors.New("write to closed connection")
	ErrDecodeIncomingMessageFailed   = errors.New("decoding the incoming message failed")
	ErrEncodeOutgoingMessageFailed   = errors.New("encoding the outgoing message failed")
	ErrDecodeIncomingContainerFailed = errors.New("decoding the incoming container failed")
	ErrDispatchFailed                = errors.New("dispatch failed")
	ErrPreflight                     = errors.New("preflight request")
)

// These are just to silence the linter.
var (
	_ = ErrNoHandler
	_ = ErrWriteToClosedConn
	_ = ErrDecodeIncomingMessageFailed
	_ = ErrEncodeOutgoingMessageFailed
	_ = ErrDecodeIncomingContainerFailed
)
