package kit

type Logger interface {
	Debugf(format string, args ...any)
	Errorf(format string, args ...any)
}

type nopLogger struct{}

var _ Logger = (*nopLogger)(nil)

func (n nopLogger) Error(_ ...any) {}

func (n nopLogger) Errorf(_ string, _ ...any) {}

func (n nopLogger) Debug(_ ...any) {}

func (n nopLogger) Debugf(_ string, _ ...any) {}
