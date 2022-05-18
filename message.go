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

func UnmarshalMessage(data []byte, m Message, enc Encoding) error {
	var err error
	switch enc {
	case Undefined:
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
	case JSON:
		if v, ok := m.(JSONUnmarshaler); ok {
			err = v.UnmarshalJSON(data)
		} else {
			err = json.Unmarshal(data, m)
		}
	case Proto:
		if v, ok := m.(ProtoUnmarshaler); ok {
			err = v.UnmarshalProto(data)
		} else {
			err = ErrUnsupportedEncoding
		}
	case Binary:
		if v, ok := m.(encoding.BinaryUnmarshaler); ok {
			err = v.UnmarshalBinary(data)
		} else {
			err = ErrUnsupportedEncoding
		}
	case Text:
		if v, ok := m.(encoding.TextUnmarshaler); ok {
			err = v.UnmarshalText(data)
		} else {
			err = ErrUnsupportedEncoding
		}
	case CustomDefined:
		if v, ok := m.(Unmarshaler); ok {
			err = v.Unmarshal(data)
		} else {
			err = ErrUnsupportedEncoding
		}
	default:
		panic("invalid encoding")
	}

	return err
}

func MarshalMessage(m Message, enc Encoding) ([]byte, error) {
	switch enc {
	case Undefined:
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
	case JSON:
		if v, ok := m.(JSONMarshaler); ok {
			return v.MarshalJSON()
		} else {
			return json.Marshal(m)
		}
	case Proto:
		if v, ok := m.(ProtoMarshaler); ok {
			return v.MarshalProto()
		} else {
			return nil, ErrUnsupportedEncoding
		}
	case Binary:
		if v, ok := m.(encoding.BinaryMarshaler); ok {
			return v.MarshalBinary()
		} else {
			return nil, ErrUnsupportedEncoding
		}
	case Text:
		if v, ok := m.(encoding.TextMarshaler); ok {
			return v.MarshalText()
		} else {
			return nil, ErrUnsupportedEncoding
		}
	case CustomDefined:
		if v, ok := m.(Marshaler); ok {
			return v.Marshal()
		} else {
			return nil, ErrUnsupportedEncoding
		}
	default:
		panic("invalid encoding")
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
