package kit

type Logger interface {
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type nopLogger struct{}

var _ Logger = (*nopLogger)(nil)

func (n nopLogger) Error(_ ...interface{}) {}

func (n nopLogger) Errorf(_ string, _ ...interface{}) {}

func (n nopLogger) Debug(_ ...interface{}) {}

func (n nopLogger) Debugf(_ string, _ ...interface{}) {}
