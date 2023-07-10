package kit

import (
	"github.com/clubpay/ronykit/kit/internal/json"
	"github.com/goccy/go-reflect"
)

type (
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
	// Message is a generic interface for all messages. Message MUST BE serializable.
	// It could implement one or many of the following interfaces:
	// 	- Marshaler
	// 	- Unmarshaler
	// 	- JSONMarshaler
	// 	- JSONUnmarshaler
	// 	- ProtoMarshaler
	// 	- ProtoUnmarshaler
	// 	- encoding.BinaryMarshaler
	// 	- encoding.BinaryUnmarshaler
	// 	- encoding.TextMarshaler
	// 	- encoding.TextUnmarshaler
	Message            any
	MessageFactoryFunc func() Message
)

func CreateMessageFactory(in Message) MessageFactoryFunc {
	switch {
	case in == nil:
		fallthrough
	case reflect.Indirect(reflect.ValueOf(in)).Type() == reflect.TypeOf(RawMessage{}):
		return func() Message {
			return RawMessage{}
		}
	}

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

type MessageMarshaler interface {
	Marshal(m any) ([]byte, error)
	Unmarshal(data []byte, m any) error
}

func SetCustomMarshaler(mm MessageMarshaler) {
	defaultMessageMarshaler = mm
}

var (
	jsonMarshaler                            = json.NewMarshaler()
	defaultMessageMarshaler MessageMarshaler = jsonMarshaler
)

func UnmarshalMessage(data []byte, m Message) error {
	return defaultMessageMarshaler.Unmarshal(data, m)
}

func unmarshalMessageX(data []byte, m Message) {
	err := jsonMarshaler.Unmarshal(data, m)
	if err != nil {
		panic(err)
	}
}

func MarshalMessage(m Message) ([]byte, error) {
	switch v := m.(type) {
	case RawMessage:
		return v, nil
	default:
		return defaultMessageMarshaler.Marshal(m)
	}
}

func marshalMessageX(m Message) []byte {
	switch v := m.(type) {
	case RawMessage:
		return v
	}
	data, err := jsonMarshaler.Marshal(m)
	if err != nil {
		panic(err)
	}

	return data
}

// RawMessage is a byte slice which could be used as a Message.
// This is helpful for raw data messages.
type RawMessage []byte

func (rm *RawMessage) Marshal() ([]byte, error) {
	return *rm, nil
}

func (rm *RawMessage) CopyFrom(in []byte) {
	*rm = append(*rm, in...)
}

// ErrorMessage is a special kind of Message which is also an error.
type ErrorMessage interface {
	GetCode() int
	GetItem() string
	Message
	error
}

// EmptyMessage is a special kind of Message which is empty.
type EmptyMessage struct{}

type JSONMessage = json.RawMessage
