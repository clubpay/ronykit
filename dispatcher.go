package ronykit

type (
	WriteFunc    func(conn Conn, e *Envelope) error
	ExecuteFunc  func(wf WriteFunc, handlers ...Handler)
	DispatchFunc func(ctx *Context, execFunc ExecuteFunc) error
)

type Dispatcher interface {
	Dispatch(conn Conn, in []byte) (DispatchFunc, error)
}
