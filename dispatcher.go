package ronykit

type (
	FlushFunc    func(m Message) error
	ExecuteFunc  func(m Message, flush FlushFunc, handlers ...Handler)
	DispatchFunc func(ctx *Context, execFunc ExecuteFunc) error
)

type Dispatcher interface {
	Dispatch(conn Conn, streamID int64, in []byte) DispatchFunc
}

type Message interface {
	Marshal() ([]byte, error)
}
