package ronykit

import (
	"reflect"
	"sync"

	"github.com/clubpay/ronykit/utils"
)

var envelopePool = &sync.Pool{}

type Walker interface {
	Walk(f func(k, v string) bool)
}

type EnvelopeHdr map[string]string

type Envelope interface {
	SetHdr(key, value string) Envelope
	SetHdrWalker(walker Walker) Envelope
	SetHdrMap(kv map[string]string) Envelope
	GetHdr(key string) string
	WalkHdr(f func(key, val string) bool) Envelope
	SetMsg(msg Message) Envelope
	GetMsg() Message
	// Send writes the envelope to the connection based on the Gateway specification.
	// You **MUST NOT** use the Envelope after calling this method.
	// You **MUST NOT** call this function more than once.
	// You **MUST NOT** call this method on incoming envelopes. i.e. the Envelope that you
	// get from Context.In
	Send()
}

// Envelope is an envelope around Message in RonyKIT. Envelopes are created internally
// by the RonyKIT framework, and provide the abstraction which Bundle implementations could
// take advantage of. For example in std/fasthttp Envelope headers translate from/to http
// request/response headers.
type envelopeImpl struct {
	ctx  *Context
	conn Conn
	kvl  utils.SpinLock
	kv   EnvelopeHdr
	m    Message

	// outgoing identity the Envelope if it is able to send
	outgoing bool

	// reuse identifies if Envelope is going to be reused by pushing to the pool.
	reuse bool
	p     *sync.Pool
}

var _ Envelope = (*envelopeImpl)(nil)

func newEnvelope(ctx *Context, conn Conn, outgoing bool) *envelopeImpl {
	e, ok := envelopePool.Get().(*envelopeImpl)
	if !ok {
		e = &envelopeImpl{
			kv:    EnvelopeHdr{},
			p:     envelopePool,
			reuse: true,
		}
	}

	for k, v := range ctx.hdr {
		e.kv[k] = v
	}

	e.ctx = ctx
	e.conn = conn
	e.outgoing = outgoing

	return e
}

func (e *envelopeImpl) release() {
	if !e.reuse {
		return
	}

	for k := range e.kv {
		delete(e.kv, k)
	}
	e.m = nil
	e.ctx = nil
	e.conn = nil

	e.p.Put(e)
}

func (e *envelopeImpl) dontReuse() {
	e.reuse = false
}

func (e *envelopeImpl) SetHdr(key, value string) Envelope {
	e.kvl.Lock()
	e.kv[key] = value
	e.kvl.Unlock()

	return e
}

func (e *envelopeImpl) SetHdrWalker(walker Walker) Envelope {
	e.kvl.Lock()
	walker.Walk(func(k, v string) bool {
		e.kv[k] = v

		return true
	})
	e.kvl.Unlock()

	return e
}

func (e *envelopeImpl) SetHdrMap(kv map[string]string) Envelope {
	e.kvl.Lock()
	for k, v := range kv {
		e.kv[k] = v
	}
	e.kvl.Unlock()

	return e
}

func (e *envelopeImpl) GetHdr(key string) string {
	e.kvl.Lock()
	v := e.kv[key]
	e.kvl.Unlock()

	return v
}

func (e *envelopeImpl) WalkHdr(f func(key string, val string) bool) Envelope {
	e.kvl.Lock()
	for k, v := range e.kv {
		if !f(k, v) {
			break
		}
	}
	e.kvl.Unlock()

	return e
}

func (e *envelopeImpl) SetMsg(msg Message) Envelope {
	e.m = msg

	return e
}

func (e *envelopeImpl) GetMsg() Message {
	if e.m == nil {
		return nil
	}

	return e.m
}

func (e *envelopeImpl) Send() {
	if e.conn == nil {
		panic("BUG!! do not call Send on nil conn, maybe called multiple times ?!")
	}

	if !e.outgoing {
		panic("BUG!! do not call Send on incoming envelope")
	}

	// run the modifiers in LIFO order
	modifiersCount := len(e.ctx.modifiers) - 1
	for idx := range e.ctx.modifiers {
		e.ctx.modifiers[modifiersCount-idx](e)
	}

	// Use WriteFunc to write the Envelope into the connection
	e.ctx.Error(e.ctx.wf(e.conn, e))

	// Release the envelope
	e.release()
}

type (
	// Modifier is a function which can modify the outgoing Envelope before sending it to the
	// client. Modifier only applies to outgoing envelopes.
	Modifier  func(envelope Envelope)
	Marshaler interface {
		Marshal() ([]byte, error)
	}
	Unmarshaler interface {
		Unmarshal([]byte) error
	}
	JSONMarshaler interface {
		MarshalJSON() ([]byte, error)
	}
	JSONUnmarshaler interface {
		UnmarshalJSON([]byte) error
	}
	ProtoMarshaler interface {
		MarshalProto() ([]byte, error)
	}
	ProtoUnmarshaler interface {
		UnmarshalProto([]byte) error
	}
	Message            interface{}
	MessageFactoryFunc func() Message
)

func CreateMessageFactory(in Message) MessageFactoryFunc {
	var ff MessageFactoryFunc

	reflect.ValueOf(&ff).Elem().Set(
		reflect.MakeFunc(
			reflect.TypeOf(ff),
			func(args []reflect.Value) (results []reflect.Value) {
				return []reflect.Value{reflect.New(reflect.TypeOf(in).Elem())}
			},
		),
	)

	return ff
}

// RawMessage is a bytes slice which could be used as Message. This is helpful for
// raw data messages.
type RawMessage []byte

func (rm RawMessage) Marshal() ([]byte, error) {
	return rm, nil
}

// ErrorMessage is a special kind of Message which is also an error.
type ErrorMessage interface {
	Message
	error
}
