package ronykit

type (
	Modifier     func(envelope *Envelope)
	WriteFunc    func(conn Conn, e *Envelope, modifiers ...Modifier) error
	ExecuteFunc  func(wf WriteFunc, handlers ...Handler)
	DispatchFunc func(ctx *Context, execFunc ExecuteFunc) error
)

type Dispatcher interface {
	Dispatch(conn Conn, in []byte) (DispatchFunc, error)
}
