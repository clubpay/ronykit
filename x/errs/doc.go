// Package errs provides an opinionated error dialect on top of
// ronykit's core errors package (rony/errs), for services that want a
// stable, client-facing error contract without re-inventing conventions.
//
// The wire contract carried by every *core.Error is fixed and small:
//
//	{"code": <int>, "codeName": "<grpc name>", "item": "<stable token>", "details": <structured>}
//
// Roles of each field:
//
//   - Code: a gRPC-style code; the edge derives the HTTP status from it.
//   - Item: a stable, machine-readable token (conventionally
//     SCREAMING_SNAKE, namespaced by area, e.g. TRIGGER_GITHUB_ORG_REQUIRED).
//     Clients may branch on it; never rename a shipped item.
//   - Details: client-visible structured context. Use Reason for
//     human-readable text, or any type implementing core.ErrDetails.
//   - Meta: internal diagnostics only; it never leaves the process.
//     Annotate markers ride here.
//   - Cause chain: internal. Powers errors.Is/As and trace logs
//     (ErrorMessage). It is NOT serialized, so the boundary error's Item is
//     the only message a client ever sees.
//
// Layer conventions:
//
//   - Domain: declare sentinels with Sentinel, and fail with Specify/
//     Specifyf so the sentinel stays on the cause chain (errors.Is keeps
//     working) while the error carries a specific item.
//   - App: mark failures with Annotate — it changes nothing on the wire.
//   - API: normalize every returned error with Boundary. It preserves the
//     richest *core.Error identity in the chain and escapes truly unknown
//     errors to Internal with a safe fallback item.
//
// See the package README for the full contract and worked examples.
package errs
