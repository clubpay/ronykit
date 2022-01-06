package ronykit

import "mime/multipart"

// Conn represents a connection.
type Conn interface {
	ConnID() uint64
	ClientIP() string
	Write(data []byte) (int, error)
	Stream() bool
	Walk(func(key string, val string) bool)
	Get(key string) string
	Set(key string, val string)
}

// REST could be implemented by Gateway, so in Dispatcher user can check if Conn also implements
// REST then it has more information about the REST request.
type REST interface {
	Conn
	GetRequestURI() string
	GetMethod() string
	GetPath() string
	Form() (*multipart.Form, error)
	SetStatusCode(code int)
}
