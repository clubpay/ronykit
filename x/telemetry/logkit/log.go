package logkit

import (
	"go.uber.org/zap/zapcore"
)

/*
   Creation Time: 2021 - Sep - 01
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
*/

const (
	DebugLevel = zapcore.DebugLevel
	InfoLevel  = zapcore.InfoLevel
	WarnLevel  = zapcore.WarnLevel
	ErrorLevel = zapcore.ErrorLevel
	PanicLevel = zapcore.PanicLevel
	FatalLevel = zapcore.FatalLevel
)

type (
	Level           = zapcore.Level
	Field           = zapcore.Field
	Entry           = zapcore.Entry
	FieldType       = zapcore.FieldType
	CheckedEntry    = zapcore.CheckedEntry
	DurationEncoder = zapcore.DurationEncoder
	CallerEncoder   = zapcore.CallerEncoder
	LevelEncoder    = zapcore.LevelEncoder
	TimeEncoder     = zapcore.TimeEncoder
	Encoder         = zapcore.Encoder
	Core            = zapcore.Core
	Hook            = func(entry Entry) error
)

var (
	DefaultLogger *Logger
	NopLogger     *Logger
)

func init() {
	DefaultLogger = New(
		WithSkipCaller(1),
	)

	NopLogger = newNOP()
}
