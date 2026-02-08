package errmarshalling

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
)

type customErr struct {
	Msg  string
	Code int
}

func (e *customErr) Error() string {
	return e.Msg
}

type valueErr struct {
	Msg string
}

func (e valueErr) Error() string {
	return e.Msg
}

type panicErr struct{}

func (panicErr) Error() string {
	return "panic"
}

type badJSON struct{}

func (badJSON) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("nope")
}

type encodeErr struct {
	msg string
}

func (e *encodeErr) Error() string {
	return e.msg
}

func TestMarshalNil(t *testing.T) {
	if string(Marshal(nil)) != "null" {
		t.Fatalf("expected null for nil marshal")
	}
}

func TestRegisterErrorMarshallerPanicsOnNonPointer(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on non-pointer error type")
		}
	}()

	RegisterErrorMarshaller[valueErr](func(_ valueErr, _ *jsoniter.Stream) {}, func(_ valueErr, _ *jsoniter.Iterator) {})
}

func TestMarshalAndUnmarshalCustomError(t *testing.T) {
	RegisterErrorMarshaller[*customErr](
		func(e *customErr, stream *jsoniter.Stream) {
			stream.WriteObjectField(ItemKey)
			stream.WriteString(e.Msg)
			stream.WriteMore()
			stream.WriteObjectField("code")
			stream.WriteInt(e.Code)
		},
		func(e *customErr, itr *jsoniter.Iterator) {
			itr.ReadObjectCB(func(itr *jsoniter.Iterator, field string) bool {
				switch field {
				case ItemKey:
					e.Msg = itr.ReadString()
				case "code":
					e.Code = itr.ReadInt()
				default:
					itr.Skip()
				}
				return true
			})
		},
	)

	data := Marshal(&customErr{Msg: "oops", Code: 42})
	unmarshaled, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}

	var ce *customErr
	if !errors.As(unmarshaled, &ce) {
		t.Fatalf("expected customErr, got %T", unmarshaled)
	}
	if ce.Msg != "oops" || ce.Code != 42 {
		t.Fatalf("unexpected custom error: %+v", ce)
	}
}

func TestMarshalFallbackWraps(t *testing.T) {
	base := errors.New("root")
	wrapped := fmt.Errorf("wrap: %w", base)

	data := Marshal(wrapped)
	unmarshaled, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if unmarshaled.Error() != "wrap: root" {
		t.Fatalf("unexpected fallback error: %v", unmarshaled)
	}
	if uw := errors.Unwrap(unmarshaled); uw == nil || uw.Error() != "root" {
		t.Fatalf("unexpected unwrap: %v", uw)
	}
}

func TestMarshalFallbackMultiWraps(t *testing.T) {
	base1 := errors.New("one")
	base2 := errors.New("two")
	joined := errors.Join(base1, base2)

	data := Marshal(joined)
	unmarshaled, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	_ = unmarshaled.Error()

	multi, ok := unmarshaled.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("expected multi unwrap, got %T", unmarshaled)
	}
	uw := multi.Unwrap()
	if len(uw) != 2 || uw[0].Error() != "one" || uw[1].Error() != "two" {
		t.Fatalf("unexpected unwrap list: %v", uw)
	}
}

func TestMarshalRecoverFromPanic(t *testing.T) {
	RegisterErrorMarshaller[*panicErr](
		func(_ *panicErr, _ *jsoniter.Stream) {
			panic("boom")
		},
		func(_ *panicErr, _ *jsoniter.Iterator) {},
	)

	data := Marshal(&panicErr{})
	if !strings.Contains(string(data), "panic occurred while marshalling error") {
		t.Fatalf("unexpected panic marshal output: %s", string(data))
	}
}

func TestTryWriteValue(t *testing.T) {
	var buf bytes.Buffer
	stream := jsoniter.ConfigDefault.BorrowStream(&buf)
	defer jsoniter.ConfigDefault.ReturnStream(stream)

	stream.WriteObjectStart()
	stream.WriteObjectField("a")
	stream.WriteString("b")

	if err := TryWriteValue(stream, "bad", badJSON{}); err == nil {
		t.Fatal("expected error from bad json")
	}

	if err := TryWriteValue(stream, "ok", map[string]string{"x": "y"}); err != nil {
		t.Fatalf("unexpected TryWriteValue error: %v", err)
	}

	stream.WriteObjectEnd()
	_ = stream.Flush()
}

func TestUnmarshalUnknownType(t *testing.T) {
	_, err := Unmarshal([]byte(`{"@type":"missing","item":"x"}`))
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestUnmarshalErrorIterator(t *testing.T) {
	itr := jsoniter.ConfigDefault.BorrowIterator([]byte(`{"@type":"missing","item":"x"}`))
	defer jsoniter.ConfigDefault.ReturnIterator(itr)

	if err := UnmarshalError(itr); err != nil {
		t.Fatalf("expected nil on unmarshal error, got %v", err)
	}
	if itr.Error == nil {
		t.Fatal("expected iterator error")
	}
}

func TestErrorInterfaceDecoder(t *testing.T) {
	data := Marshal(errors.New("oops"))
	var err error
	if e := json.Unmarshal(data, &err); e != nil {
		t.Fatal(e)
	}
	if err == nil || err.Error() != "oops" {
		t.Fatalf("unexpected decoded error: %v", err)
	}
}

func TestMarshalErrorFallbackOnEncoderError(t *testing.T) {
	RegisterErrorMarshaller[*encodeErr](
		func(_ *encodeErr, stream *jsoniter.Stream) {
			stream.Error = errors.New("encode")
		},
		func(_ *encodeErr, _ *jsoniter.Iterator) {},
	)

	data := Marshal(&encodeErr{msg: "encode-failed"})
	if !strings.Contains(string(data), "encode") {
		t.Fatalf("unexpected fallback output: %s", string(data))
	}
}

func TestJSONMarshallerIsEmpty(t *testing.T) {
	m := &jsonMarshaller{}
	if m.IsEmpty(nil) {
		t.Fatal("expected IsEmpty to return false")
	}
}
