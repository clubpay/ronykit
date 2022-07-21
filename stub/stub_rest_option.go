package stub

type RESTOption func(ctx *restClientCtx)

func WithPreflightREST(h ...RESTPreflightHandler) RESTOption {
	return func(ctx *restClientCtx) {
		ctx.preflights = append(ctx.preflights[:0], h...)

		return
	}
}
