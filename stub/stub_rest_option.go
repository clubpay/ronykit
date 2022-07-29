package stub

type RESTOption func(ctx *restClientCtx)

// WithPreflightREST register one or many handlers to run in sequence before
// actually making requests.
func WithPreflightREST(h ...RESTPreflightHandler) RESTOption {
	return func(ctx *restClientCtx) {
		ctx.preflights = append(ctx.preflights[:0], h...)

		return
	}
}
