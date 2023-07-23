package proxy

import (
	"fmt"
)

type __Logger interface {
	Printf(format string, args ...interface{})
}

type nopLogger struct{}

func (n *nopLogger) Printf(format string, args ...interface{}) {
	// if format not end with '\n', then append it
	if format[len(format)-1] != '\n' {
		format += "\n"
	}

	fmt.Printf(format, args...)
}

func debugF(debug bool, logger __Logger, format string, args ...any) {
	if logger == nil || !debug {
		return
	}

	logger.Printf("[debug] "+format, args...)
}

func errorF(logger __Logger, format string, args ...any) {
	if logger == nil {
		return
	}

	logger.Printf("[error] "+format, args...)
}
