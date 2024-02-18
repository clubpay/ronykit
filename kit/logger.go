package kit

type Logger interface {
	Debugf(format string, args ...any)
	Errorf(format string, args ...any)
}

type NOPLogger struct{}

var _ Logger = (*NOPLogger)(nil)

func (n NOPLogger) Error(_ ...any) {}

func (n NOPLogger) Errorf(_ string, _ ...any) {}

func (n NOPLogger) Debug(_ ...any) {}

func (n NOPLogger) Debugf(_ string, _ ...any) {}
