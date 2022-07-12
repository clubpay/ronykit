package ronykit

type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

type nopLogger struct{}

func (n nopLogger) Error(args ...interface{}) {}

func (n nopLogger) Errorf(format string, args ...interface{}) {}

func (n nopLogger) Debug(args ...interface{}) {}

func (n nopLogger) Debugf(format string, args ...interface{}) {}
