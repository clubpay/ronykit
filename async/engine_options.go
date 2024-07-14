package async

type Option func(*Engine) error

type component interface {
	register(srv *Engine) error
	unregister(srv *Engine)
}

// Register is a function that takes one or more tasks/queues and registers them with a server.
// Task and Queue implement component interface.
// The function returns an Option type that can be used in NewEngine builder.
func Register(components ...component) Option {
	return func(server *Engine) error {
		for _, task := range components {
			err := task.register(server)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

// ErrFunc set the global error handler for all unhandled exceptions raised by
// components. Usually there is little we can do about these errors except to
// capture for logging purposes.
func ErrFunc(errFn func(err error)) Option {
	return func(srv *Engine) error {
		srv.errFunc = errFn

		return nil
	}
}
