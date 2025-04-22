package flow

import (
	"errors"

	"go.temporal.io/sdk/temporal"
)

// IsApplicationError return if the err is a ApplicationError
func IsApplicationError(err error) (bool, *temporal.ApplicationError) {
	var applicationError *temporal.ApplicationError
	ok := errors.As(err, &applicationError)

	return ok, applicationError
}
