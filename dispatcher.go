package ronykit

type (
	WriteFunc    func(m Message)
	ExecuteFunc  func(m Message, wf WriteFunc, handlers ...Handler)
	DispatchFunc func(ctx *Context, execFunc ExecuteFunc) error
)

type Dispatcher interface {
	Dispatch(conn Conn, streamID int64, in []byte) DispatchFunc
}

type Message interface {
	Marshal() ([]byte, error)
}
