Register global middlewares via `di.RegisterMiddleware(mw1, mw2, mw3)` in an
`init()` function, typically in the `cmd/` entrypoint package.

Standard middleware stack:

1. Logging middleware:
   - adds trace events for request/response,
   - enriches spans with HTTP semantic conventions (status code, client IP,
     user agent),
   - sets a `Trace-ID` response header,
   - marks spans as errors for status `>= 400`.
2. Base header middleware:
   - presets `Content-Type: application/json`.
3. Panic recovery middleware:
   - catches panics,
   - records the stack trace in the span,
   - returns a `500` error with a generic `TECHNICAL_PROBLEM` message.

Each middleware is a `func(ctx *kit.Context)` that calls `ctx.Next()` to
continue the chain.
