package stub

import "github.com/clubpay/ronykit/kit"

type RESTOption func(cfg *restConfig)

type restConfig struct {
	preflights []RESTPreflightHandler
	tp         kit.TracePropagator
}

// WithPreflightREST register one or many handlers to run in sequence before
// actually making requests.
func WithPreflightREST(h ...RESTPreflightHandler) RESTOption {
	return func(cfg *restConfig) {
		cfg.preflights = append(cfg.preflights[:0], h...)

		return
	}
}
