package common

type NOPLogger struct{}

func NewNopLogger() NOPLogger {
	return NOPLogger{}
}

func (n NOPLogger) Debug(args ...interface{}) {}

func (n NOPLogger) Debugf(format string, args ...interface{}) {}

func (n NOPLogger) Error(args ...interface{}) {}

func (n NOPLogger) Errorf(format string, args ...interface{}) {}
