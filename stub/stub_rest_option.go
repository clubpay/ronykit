package stub

import "github.com/clubpay/ronykit/kit"

type RESTOption func(cfg *restConfig)

type restConfig struct {
	preflights []RESTPreflightHandler
	tp         kit.TracePropagator
	hdr        map[string]string
}

// WithPreflightREST register one or many handlers to run in sequence before
// actually making requests.
func WithPreflightREST(h ...RESTPreflightHandler) RESTOption {
	return func(cfg *restConfig) {
		cfg.preflights = append(cfg.preflights[:0], h...)
	}
}

func WithHeaderMap(hdr map[string]string) RESTOption {
	return func(cfg *restConfig) {
		cfg.hdr = hdr
	}
}

func WithHeader(key, value string) RESTOption {
	return func(cfg *restConfig) {
		if cfg.hdr == nil {
			cfg.hdr = make(map[string]string)
		}

		cfg.hdr[key] = value
	}
}
