package kit

import (
	"io"
	"mime/multipart"

	"github.com/goccy/go-json"
	"github.com/goccy/go-reflect"
)

type (
	Marshaler interface {
		Marshal() ([]byte, error)
	}
	Unmarshaler interface {
		Unmarshal(data []byte) error
	}
	JSONMarshaler interface {
		MarshalJSON() ([]byte, error)
	}
	JSONUnmarshaler interface {
		UnmarshalJSON(data []byte) error
	}
	ProtoMarshaler interface {
		MarshalProto() ([]byte, error)
	}
	ProtoUnmarshaler interface {
		UnmarshalProto(data []byte) error
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
	case reflect.Indirect(reflect.ValueOf(in)).Type() == reflect.TypeOf(MultipartFormMessage{}):
		return func() Message {
			return MultipartFormMessage{}
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

type MessageCodec interface {
	Encode(m Message, w io.Writer) error
	Marshal(m any) ([]byte, error)
	Decode(m Message, r io.Reader) error
	Unmarshal(data []byte, m any) error
}

type stdCodec struct{}

// Encode marshal the Message m into the Writer w.
// NOTE: this method will add one extra \n at the end of the output, which is different from
// the output of Marshal method.
func (stdCodec) Encode(m Message, w io.Writer) error {
	return json.NewEncoder(w).EncodeWithOption(m)
}

func (stdCodec) Decode(m Message, r io.Reader) error {
	return json.NewDecoder(r).Decode(m)
}

func (jm stdCodec) Marshal(m any) ([]byte, error) {
	return json.MarshalNoEscape(m)
}

func (jm stdCodec) Unmarshal(data []byte, m any) error {
	return json.UnmarshalNoEscape(data, m)
}

func SetCustomCodec(mm MessageCodec) {
	defaultMessageCodec = mm
}

func GetMessageCodec() MessageCodec {
	return defaultMessageCodec
}

var defaultMessageCodec MessageCodec = stdCodec{}

func UnmarshalMessage(data []byte, m Message) error {
	return defaultMessageCodec.Unmarshal(data, m)
}

func MarshalMessage(m Message) ([]byte, error) {
	switch v := m.(type) {
	case RawMessage:
		return v, nil
	default:
		return defaultMessageCodec.Marshal(m)
	}
}

func EncodeMessage(m Message, w io.Writer) error {
	switch v := m.(type) {
	case RawMessage:
		_, err := w.Write(v)

		return err
	default:
		return defaultMessageCodec.Encode(v, w)
	}
}

func DecodeMessage(m Message, r io.Reader) error {
	return defaultMessageCodec.Decode(m, r)
}

// RawMessage is a byte slice which could be used as a Message.
// This is helpful for raw data messages.
// Example:
//
//	SetInput(kit.RawMessage)
type RawMessage []byte

func (rm RawMessage) Marshal() ([]byte, error) {
	return rm, nil
}

func (rm RawMessage) MarshalJSON() ([]byte, error) {
	return rm, nil
}

func (rm *RawMessage) CopyFrom(in []byte) {
	*rm = append(*rm, in...)
}

func (rm *RawMessage) CopyTo(out []byte) {
	copy(out, *rm)
}

// Clone copies the underlying byte slice into dst. It is SAFE to
// pass nil for dst.
func (rm *RawMessage) Clone(dst []byte) []byte {
	dst = append(dst, *rm...)

	return dst
}

func (rm *RawMessage) Unmarshal(data []byte) error {
	*rm = append(*rm, data...)

	return nil
}

func (rm *RawMessage) UnmarshalJSON(data []byte) error {
	*rm = append(*rm, data...)

	return nil
}

func CastRawMessage[M Message](rawMsg RawMessage) (*M, error) {
	var m M
	err := UnmarshalMessage(rawMsg, &m)
	if err != nil {
		return nil, err
	}

	return &m, nil
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

// MultipartFormMessage is a message type for multipart form data.
// This is like RawMessage a special kind of message. When you define
// them in Descriptor, your MUST NOT pass address of them like normal
// messages.
// Example:
//
//	SetInput(kit.MultipartFormMessage)
type MultipartFormMessage struct {
	frm *multipart.Form
}

func (mfm *MultipartFormMessage) SetForm(frm *multipart.Form) {
	mfm.frm = frm
}

func (mfm *MultipartFormMessage) GetForm() *multipart.Form {
	return mfm.frm
}
