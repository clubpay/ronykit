package errs

import (
	"errors"
	"net/http"

	"github.com/clubpay/ronykit/rony/errs/errmarshalling"
	jsoniter "github.com/json-iterator/go"
)

var statusToCode = map[int]ErrCode{
	200: OK,
	499: Canceled,
	500: Internal,
	400: InvalidArgument,
	401: Unauthenticated,
	403: PermissionDenied,
	404: NotFound,
	409: AlreadyExists,
	429: ResourceExhausted,
	501: Unimplemented,
	503: Unavailable,
	504: DeadlineExceeded,
}

func HTTPStatusToCode(status int) ErrCode {
	if c, ok := statusToCode[status]; ok {
		return c
	}

	return Unknown
}

func HTTPStatus(err error) int {
	code := Code(err)
	switch code {
	case OK:
		return 200
	case Canceled:
		return 499
	case Unknown:
		return 500
	case InvalidArgument:
		return 400
	case DeadlineExceeded:
		return 504
	case NotFound:
		return 404
	case AlreadyExists:
		return 409
	case PermissionDenied:
		return 403
	case ResourceExhausted:
		return 429
	case FailedPrecondition:
		return 400
	case Aborted:
		return 409
	case OutOfRange:
		return 400
	case Unimplemented:
		return 501
	case Internal:
		return 500
	case Unavailable:
		return 503
	case DataLoss:
		return 500
	case Unauthenticated:
		return 401
	default:
		return 500
	}
}

// HTTPErrorWithCode writes structured error information to w using JSON encoding.
// The given status code is used if it is non-zero, and otherwise
// it is computed with HTTPStatus.
//
// If err is nil it writes:
//
//	{"code": "ok", "message": "", "details": null}
func HTTPErrorWithCode(w http.ResponseWriter, err error, code int) {
	if code == 0 {
		code = HTTPStatus(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if err == nil {
		w.WriteHeader(code)
		_, _ = w.Write([]byte(`{
  "code": "ok",
  "message": "",
  "details": null
}
`))

		return
	}

	e := func() *Error {
		target := &Error{}
		_ = errors.As(Convert(err), &target)

		return target
	}()
	data, err2 := json.MarshalIndent(e, "", "  ")
	if err2 != nil {
		// Must be the details; drop them
		e2 := &Error{Code: e.Code, Message: e.Message}
		data, _ = json.MarshalIndent(e2, "", "  ")
	}
	w.WriteHeader(code)
	// nosemgrep
	_, _ = w.Write(data)
}

// writeErrorFieldsToInternalStream writes the error fields to the given stream
// for passing between running Encore services.
//
// Note we do not marshal the Details object as we would need to allow
// for reflection loading of the type by name, which is not safe and could
// lead to arbitrary code execution.
func writeErrorFieldsToInternalStream(e *Error, stream *jsoniter.Stream) {
	stream.WriteObjectField("code")
	stream.WriteInt(int(e.Code))

	stream.WriteMore()
	stream.WriteObjectField(errmarshalling.MessageKey)
	stream.WriteString(e.Message)

	if len(e.Meta) > 0 {
		if err := errmarshalling.TryWriteValue(stream, "meta", e.Meta); err != nil {
			// Only report the error in the JSON stream
			// don't error out the whole marshal as it's critical
			// that we marshal the error.
			stream.WriteMore()
			stream.WriteObjectField("meta_marshal_error")
			stream.WriteString(err.Error())
		}
	}

	if e.underlying != nil {
		stream.WriteMore()
		stream.WriteObjectField(errmarshalling.WrappedKey)
		stream.WriteVal(e.underlying)
	}
}

func unmarshalFromInternalIterator(e *Error, itr *jsoniter.Iterator) {
	itr.ReadObjectCB(func(itr *jsoniter.Iterator, field string) bool {
		switch field {
		case "code":
			e.Code = ErrCode(itr.ReadInt())
		case errmarshalling.MessageKey:
			e.Message = itr.ReadString()
		case "meta":
			itr.ReadVal(&e.Meta)
		case errmarshalling.WrappedKey:
			e.underlying = errmarshalling.UnmarshalError(itr)
		default:
			itr.Skip()
		}

		return true
	})
}

func init() {
	errmarshalling.RegisterErrorMarshaller(writeErrorFieldsToInternalStream, unmarshalFromInternalIterator)
}
