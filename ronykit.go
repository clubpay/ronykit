package ronykit

import (
	"context"
	"mime/multipart"
)

// Bundle is the pluggable component of the EdgeServer.
type Bundle interface {
	// Start starts the gateway to accept connections.
	Start(ctx context.Context) error
	// Shutdown shuts down the gateway gracefully.
	Shutdown(ctx context.Context) error
	// Register registers the route in the Bundle. This is how Bundle get information
	// about the services and their contracts.
	Register(serviceName, contractID string, sel RouteSelector, input Message)
}

type (
	ExecuteArg struct {
		WriteFunc
		ServiceName string
		ContractID  string
		Route       string
	}
	WriteFunc   func(conn Conn, e Envelope) error
	ExecuteFunc func(arg ExecuteArg)
)

var NoExecuteArg = ExecuteArg{}

// Conn represents a connection between EdgeServer and client.
type Conn interface {
	ConnID() uint64
	ClientIP() string
	Write(data []byte) (int, error)
	Stream() bool
	Walk(func(key string, val string) bool)
	Get(key string) string
	Set(key string, val string)
}

// RESTConn could be implemented by Gateway, so in Dispatcher user can check if Conn also implements
// RESTConn then it has more information about the RESTConn request.
type RESTConn interface {
	Conn
	GetMethod() string
	GetHost() string
	// GetRequestURI returns uri without Method and Host
	GetRequestURI() string
	// GetPath returns uri without Method, Host and Query parameters.
	GetPath() string
	Form() (*multipart.Form, error)
	SetStatusCode(code int)
	Redirect(code int, url string)
}
