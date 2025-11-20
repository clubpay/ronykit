package common

import (
	"fmt"

	"github.com/clubpay/ronykit/kit"
)

type stdLogger struct{}

func NewStdLogger() kit.Logger {
	return stdLogger{}
}

var _ kit.Logger = (*stdLogger)(nil)

func (s stdLogger) Debugf(format string, args ...any) {
	fmt.Printf("DEBUG: %s\n", fmt.Sprintf(format, args...)) //nolint:forbidigo
}

func (s stdLogger) Errorf(format string, args ...any) {
	fmt.Printf("ERROR: %s\n", fmt.Sprintf(format, args...)) //nolint:forbidigo
}
