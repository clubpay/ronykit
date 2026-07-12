---

keywords:
- rest
- http
- api
applies_to_files:
- api

---

Focus on api/service.go contracts, request validation, and route semantics. Generate API docs with x/apidoc.

For passthrough HTTP/WebSocket proxy routes, use `rony.WithRelay` and read `architecture/handler-relay` — not `WithUnary`.

## File-Level Hint

Define explicit contracts/DTOs and validate request input at API boundary. Generate docs with x/apidoc.
