package common

import "github.com/clubpay/ronykit/kit"

type NOPLogger struct{}

func NewNopLogger() NOPLogger {
	return NOPLogger{}
}

var _ kit.Logger = (*NOPLogger)(nil)

func (n NOPLogger) Debug(_ ...any) {}

func (n NOPLogger) Debugf(_ string, _ ...any) {}

func (n NOPLogger) Error(_ ...any) {}

func (n NOPLogger) Errorf(_ string, _ ...any) {}
