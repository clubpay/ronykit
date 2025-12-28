package proxy

import (
	"fmt"
	"os"
)

type __Logger interface {
	Printf(format string, args ...any)
}

type nopLogger struct{}

func (n *nopLogger) Printf(format string, args ...any) {
	// if format not end with '\n', then append it
	if format[len(format)-1] != '\n' {
		format += "\n"
	}

	_, _ = fmt.Fprintf(os.Stdout, format, args...)
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
