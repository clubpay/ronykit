package ronykit

import (
	"encoding"
	"errors"

	"github.com/goccy/go-json"
	"github.com/goccy/go-reflect"
)

var ErrUnsupportedEncoding = errors.New("unsupported encoding")

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

func UnmarshalMessage(data []byte, m Message) error {
	var err error
	switch v := m.(type) {
	case Unmarshaler:
		err = v.Unmarshal(data)
	case ProtoUnmarshaler:
		err = v.UnmarshalProto(data)
	case JSONUnmarshaler:
		err = v.UnmarshalJSON(data)
	case encoding.BinaryUnmarshaler:
		err = v.UnmarshalBinary(data)
	case encoding.TextUnmarshaler:
		err = v.UnmarshalText(data)
	default:
		err = json.Unmarshal(data, m)
	}

	return err
}

func MarshalMessage(m Message) ([]byte, error) {
	switch v := m.(type) {
	case Marshaler:
		return v.Marshal()
	case ProtoMarshaler:
		return v.MarshalProto()
	case JSONMarshaler:
		return v.MarshalJSON()
	case encoding.BinaryMarshaler:
		return v.MarshalBinary()
	case encoding.TextMarshaler:
		return v.MarshalText()
	default:
		return json.Marshal(m)
	}
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
