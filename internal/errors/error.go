package errors

type ErrFunc func(v ...interface{}) error
