package ronykit

import "mime/multipart"

// Conn represents a connection.
type Conn interface {
	ConnID() uint64
	ClientIP() string
	Write(streamID int64, data []byte) error
	Stream() bool
	Walk(func(key string, val interface{}) bool)
	Get(key string) interface{}
	Set(key string, val interface{})
}

// REST could be implemented by Gateway, so in Dispatcher user can check if Conn also implements
// REST then it has more information about the REST request.
type REST interface {
	GetMethod() string
	GetPath() string
	Form() (*multipart.Form, error)
}
