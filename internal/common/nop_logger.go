package common

import "github.com/clubpay/ronykit"

type nopLogger struct{}

func NewNopLogger() ronykit.Logger {
	return nopLogger{}
}

func (n nopLogger) Debug(args ...interface{}) {
	return
}

func (n nopLogger) Debugf(format string, args ...interface{}) {
	return
}

func (n nopLogger) Error(args ...interface{}) {
	return
}

func (n nopLogger) Errorf(format string, args ...interface{}) {
	return
}
