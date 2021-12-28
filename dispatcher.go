package ronykit

type (
	WriteFunc    func(m Message, ctxKey ...string)
	ExecuteFunc  func(m Message, wf WriteFunc, handlers ...Handler)
	DispatchFunc func(ctx *Context, execFunc ExecuteFunc) error
)

type Dispatcher interface {
	Dispatch(conn Conn, in []byte) (DispatchFunc, error)
}

type Message interface {
	Marshal() ([]byte, error)
}
