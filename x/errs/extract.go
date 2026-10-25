package errs

import (
	"errors"

	core "github.com/clubpay/ronykit/rony/errs"
)

// Reason is the standard client-visible Details payload for errors whose
// only extra context is a human-readable explanation. Use a typed struct
// instead when the client needs to branch on the payload.
type Reason struct {
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// ErrDetails marks Reason as a core.ErrDetails payload.
func (Reason) ErrDetails() {}

// Item returns the stable wire token of the outermost *core.Error in the
// chain, or "" when the chain carries none.
func Item(err error) string {
	e := &core.Error{}
	if errors.As(err, &e) {
		return e.Item
	}

	return ""
}

// IsCode reports the code the edge would turn into an HTTP status: the
// outermost *core.Error's code, or Unknown for plain errors.
func IsCode(err error, code ErrCode) bool {
	return core.Code(err) == code
}

// DetailsAs extracts typed details from the outermost *core.Error in the
// chain. It works in-process only; details that arrived over HTTP decode as
// map[string]any and never match T.
func DetailsAs[T ErrDetails](err error) (T, bool) {
	var zero T

	e := &core.Error{}
	if !errors.As(err, &e) {
		return zero, false
	}

	d, ok := e.Details.(T)
	if !ok {
		return zero, false
	}

	return d, true
}
