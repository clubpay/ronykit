package ronykit

type (
	WriteFunc    func(e *Envelope) error
	ExecuteFunc  func(e *Envelope, wf WriteFunc, handlers ...Handler)
	DispatchFunc func(ctx *Context, execFunc ExecuteFunc) error
)

type Dispatcher interface {
	Dispatch(conn Conn, in []byte) (DispatchFunc, error)
}

type Message interface {
	Marshal() ([]byte, error)
}
