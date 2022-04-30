package ronykit

import "mime/multipart"

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
	GetHost() string
	GetRequestURI() string
	GetMethod() string
	GetPath() string
	Form() (*multipart.Form, error)
	SetStatusCode(code int)
	Redirect(code int, url string)
}
