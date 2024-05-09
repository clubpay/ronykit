package kit

// Conn represents a connection between EdgeServer and client.
type Conn interface {
	ConnID() uint64
	ClientIP() string
	Write(data []byte) (int, error)
	WriteEnvelope(e *Envelope) error
	Stream() bool
	Walk(fn func(key string, val string) bool)
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
	SetStatusCode(code int)
	Redirect(code int, url string)
	WalkQueryParams(fn func(key string, val string) bool)
}
