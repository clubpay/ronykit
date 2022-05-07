package ronykit

import "context"

// Bundle is the pluggable component of the EdgeServer.
type Bundle interface {
	// Start starts the gateway to accept connections.
	Start(ctx context.Context) error
	// Shutdown shuts down the gateway gracefully.
	Shutdown(ctx context.Context) error
	// Register registers the svc Service in the Bundle. This is how Bundle get information
	// about the services and their contracts.
	Register(svc Service)
}

type Dispatcher interface {
	// Dispatch receives the messages from external clients and returns DispatchFunc
	// The user of the Gateway does not need to implement this. If you are using some standard bundles
	// like std/gateway/fasthttp or std/gateway/fastws then all the implementation is taken care of.
	Dispatch(ctx *Context, in []byte, execFunc ExecuteFunc) error
}

type (
	ExecuteArg struct {
		WriteFunc
		HandlerFuncChain
	}
	WriteFunc   func(conn Conn, e *Envelope) error
	ExecuteFunc func(ctx *Context, arg ExecuteArg)
)
