package proxy

import (
	"errors"

	"github.com/valyala/fasthttp"
)

// errClosed is the error resulting if the pool is closed via pool.Close().
var errClosed = errors.New("pool is closed")

// Proxier can be HTTP or WebSocket proxier
type Proxier interface {
	ServeHTTP(ctx *fasthttp.RequestCtx)
	SetClient(addr string) Proxier
	Reset()
	Close()
}

// Pool interface ...
// this interface ref to: https://github.com/fatih/pool/blob/master/pool.go
type Pool interface {
	// Get returns a new ReverseProxy from the pool.
	Get(route string) (*ReverseProxy, error)

	// Put Reseting the ReverseProxy puts it back to the Pool.
	Put(p *ReverseProxy) error

	// Close closes the pool and all its connections. After Close() the pool is
	// no longer usable.
	Close()

	// Len returns the current number of connections of the pool.
	Len() int
}
