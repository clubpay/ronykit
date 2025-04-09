package log

import (
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	io.Writer
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Printf(format string, args ...interface{})
	FileLogger(filePath string) *fileLogger
}

var _ Logger = &stdLogger{}

type stdLogger struct {
	zapLogger *zap.SugaredLogger
	stdOutWriter
}

func (s *stdLogger) Debugf(format string, args ...interface{}) {
	s.zapLogger.Debugf(format, args...)
}

func (s *stdLogger) Infof(format string, args ...interface{}) {
	s.zapLogger.Infof(format, args...)
}

func (s *stdLogger) Warnf(format string, args ...interface{}) {
	s.zapLogger.Warnf(format, args...)
}

func (s *stdLogger) Printf(format string, args ...interface{}) {
	s.zapLogger.Debugf(format, args...)
}

func (s *stdLogger) FileLogger(filePath string) *fileLogger {
	l := &fileLogger{}
	f, err := os.Create(filePath)
	if err != nil {
		s.Warnf("got error on creating file logger: %v", err)

		return l
	}
	l.f = f
	l.l = s

	return l
}

type stdOutWriter struct{}

func (s stdOutWriter) Write(p []byte) (n int, err error) {
	if len(p) > 1 {
		_, _ = os.Stdout.WriteString("||  ")
	}

	return os.Stdout.Write(p)
}

func New(lvl int) Logger {
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			MessageKey:       "msg",
			LevelKey:         "lvl",
			TimeKey:          "time",
			NameKey:          "name",
			CallerKey:        "",
			FunctionKey:      "fn",
			EncodeLevel:      zapcore.CapitalColorLevelEncoder,
			EncodeTime:       zapcore.TimeEncoderOfLayout(time.Kitchen),
			EncodeDuration:   zapcore.StringDurationEncoder,
			EncodeCaller:     zapcore.ShortCallerEncoder,
			ConsoleSeparator: "   ",
		}),
		os.Stdout,
		zap.NewAtomicLevelAt(zapcore.Level(lvl)),
	)

	return &stdLogger{
		zapLogger: zap.New(core).Sugar(),
	}
}

type fileLogger struct {
	l *stdLogger
	f *os.File
}

func (f fileLogger) Printf(format string, args ...interface{}) {
	data := []byte(fmt.Sprintf(format, args...))
	if f.f != nil {
		_, _ = f.f.Write(data)
	}

	_, _ = f.l.Write(data)
}

func (f fileLogger) Close() {
	_ = f.f.Close()
}
