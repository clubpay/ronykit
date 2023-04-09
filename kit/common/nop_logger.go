package common

import "github.com/clubpay/ronykit/kit"

type NOPLogger struct{}

func NewNopLogger() NOPLogger {
	return NOPLogger{}
}

var _ kit.Logger = (*NOPLogger)(nil)

func (n NOPLogger) Debug(_ ...interface{}) {}

func (n NOPLogger) Debugf(_ string, _ ...interface{}) {}

func (n NOPLogger) Error(_ ...interface{}) {}

func (n NOPLogger) Errorf(_ string, _ ...interface{}) {}
