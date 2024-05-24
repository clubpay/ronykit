package common

import "github.com/clubpay/ronykit/kit"

type nopLogger struct{}

func NewNopLogger() kit.Logger {
	return nopLogger{}
}

var _ kit.Logger = (*nopLogger)(nil)

func (n nopLogger) Debug(_ ...any) {}

func (n nopLogger) Debugf(_ string, _ ...any) {}

func (n nopLogger) Error(_ ...any) {}

func (n nopLogger) Errorf(_ string, _ ...any) {}
