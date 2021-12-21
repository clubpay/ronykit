package ronykit

// Gateway defines the gateway interface where clients could connect
// and communicate with the Edge servers
type Gateway interface {
	// Start starts the gateway to accept connections.
	Start()
	// Shutdown shuts down the gateway gracefully.
	Shutdown()
	// Subscribe will be called by the EdgeServer the gateway. These delegate functions
	// must be called by the Gateway implementation.
	//
	// NOTE: This func will be called only once and before calling Start function.
	Subscribe(d GatewayDelegate)
}

type GatewayDelegate interface {
	// OnOpen must be called whenever a new connection is established.
	OnOpen(c GatewayConn)
	// OnClose must be called whenever the connection is gone.
	OnClose(connID uint64)
	// OnMessage must be called whenever a new message arrives.
	OnMessage(c GatewayConn, streamID int64, msg []byte) error
}

type GatewayConn interface {
	ConnID() uint64
	ClientIP() string
	Write(streamID int64, data []byte) error
	Stream() bool
	Walk(func(key string, val interface{}) bool)
	Get(key string) interface{}
	Set(key string, val interface{})
}
