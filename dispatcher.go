package ronykit

type (
	WriteFunc      func(conn Conn, e *Envelope) error
	ExecuteFunc    func(wf WriteFunc, handlers ...Handler)
	DispatchFunc   func(ctx *Context, execFunc ExecuteFunc) error
	MessageFactory func() Message
	Modifier       func(envelope *Envelope)
)

type Dispatcher interface {
	Dispatch(conn Conn, in []byte) (DispatchFunc, error)
}

type Message interface {
	Marshal() ([]byte, error)
}

type RawMessage []byte

func (rm RawMessage) Marshal() ([]byte, error) {
	return rm, nil
}
