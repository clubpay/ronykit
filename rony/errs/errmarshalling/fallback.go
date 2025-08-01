package errmarshalling

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
)

func init() {
	marshaller := &jsonMarshaller{
		unmarshal: fallbackUnmarshal,
		decoder: func(_ unsafe.Pointer, iter *jsoniter.Iterator) {
			err := fallbackUnmarshal(iter)
			fmt.Println("here with ", err) //nolint:forbidigo
		},
	}
	serdeByName["error"] = marshaller
}

func fallbackUnmarshal(itr *jsoniter.Iterator) error {
	var (
		errMsg      string
		wrapped     error
		wrappedList []error
	)

	itr.ReadObjectCB(func(itr *jsoniter.Iterator, field string) bool {
		switch field {
		case ItemKey:
			errMsg = itr.ReadString()
		case WrappedKey:
			switch itr.WhatIsNext() {
			case jsoniter.ArrayValue:
				itr.ReadArrayCB(func(itr *jsoniter.Iterator) bool {
					wrappedList = append(wrappedList, UnmarshalError(itr))

					return true
				})
			case jsoniter.ObjectValue:
				wrapped = UnmarshalError(itr)
			default:
				itr.ReportError("unmarshal", "expected array or object")
				itr.Skip()
			}
		default:
			itr.Skip()
		}

		return true
	})

	switch {
	case len(wrappedList) > 0:
		return &fallbackMultiWrapErr{
			item:    errMsg,
			wrapped: wrappedList,
		}
	case wrapped != nil:
		return &fallbackSingleWrapErr{
			item:    errMsg,
			wrapped: wrapped,
		}
	default:
		return errors.New(errMsg)
	}
}

func createFallbackEncoder(typ reflect2.Type) jsoniter.ValEncoder {
	return &jsonMarshaller{
		encoder: func(ptr unsafe.Pointer, stream *jsoniter.Stream) {
			pointerToVal := typ.PackEFace(ptr)                        // *[error type]
			iface := reflect.ValueOf(pointerToVal).Elem().Interface() // [error type]
			err := iface.(error)                                      // error

			// check if we have a custom marshaller for this type
			// if we do then use it.
			//
			// this path can occur if the field is strongly typed as `error`
			_, found := serdeByType[reflect2.TypeOf(iface)]
			if found {
				// this will go via the custom marshaller now
				stream.WriteVal(iface)

				return
			}

			stream.WriteObjectStart()

			stream.WriteObjectField(TypeKey)
			stream.WriteString("error")
			stream.WriteMore()

			stream.WriteObjectField(ItemKey)
			stream.WriteString(err.Error())

			if err, ok := err.(interface{ Unwrap() error }); ok {
				wrapped := err.Unwrap()
				if wrapped != nil {
					stream.WriteMore()
					stream.WriteObjectField(WrappedKey)
					stream.WriteVal(wrapped)
				}
			}

			if err, ok := err.(interface{ Unwrap() []error }); ok {
				wrapped := err.Unwrap()
				if len(wrapped) > 0 {
					stream.WriteMore()
					stream.WriteObjectField(WrappedKey)
					stream.WriteVal(wrapped)
				}
			}

			stream.WriteObjectEnd()
		},
	}
}

// fallbackSingleWrapErr is a fallback error type that wraps a single error.
type fallbackSingleWrapErr struct {
	item    string
	wrapped error
}

var _ error = (*fallbackSingleWrapErr)(nil)

func (f *fallbackSingleWrapErr) Error() string {
	return f.item
}

func (f *fallbackSingleWrapErr) Unwrap() error {
	return f.wrapped
}

// fallbackMultiWrapErr is a fallback error type that wraps multiple errors.
type fallbackMultiWrapErr struct {
	item    string
	wrapped []error
}

var _ error = (*fallbackMultiWrapErr)(nil)

func (f *fallbackMultiWrapErr) Error() string {
	return f.item
}

func (f *fallbackMultiWrapErr) Unwrap() []error {
	return f.wrapped
}
