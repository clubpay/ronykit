---
name: ronykit-framework
description: Build and modify services using the RonyKit Go framework, including rony, kit, ronyup scaffolding, gateways, clusters, and stubs. Use when the user mentions RonyKit/Rony, EdgeServer, contracts, ronyup, MCP scaffolding, or asks to add API handlers/services in this repository style.
---

# RonyKit Framework

## Quick Start

When implementing work in a RonyKit project:

1. Determine abstraction level:
   - Use `rony/` for batteries-included service development.
   - Use `kit/` for low-level transport/gateway/cluster customization.
2. Respect module boundaries in the Go workspace (`go.work` with many `go.mod` files).
3. Prefer focused tests in the touched module, then broader checks if needed.

## Repository Map

- `rony/`: high-level server APIs and typed request contexts
- `kit/`: low-level core primitives (`EdgeServer`, contracts, routing/context internals)
- `ronyup/`: scaffolding CLI and MCP server (`ronyup mcp`)
- `std/gateways/`: gateway implementations (`fasthttp`, `silverhttp`, `fastws`, `mcp`)
- `std/clusters/`: cluster backends (`rediscluster`, `p2pcluster`)
- `stub/`: client stub generation (Go and TypeScript)
- `flow/`: workflow helpers and Temporal integration
- `x/`: optional utilities (DI, telemetry, cache, i18n, apidoc, etc.)

## Implementation Workflow

1. Locate the target module and read nearby docs/README first.
2. Keep changes scoped; avoid cross-module refactors unless asked.
3. Preserve existing architecture: Service -> Contract -> Handler chain.
4. Keep transport/routing behavior explicit:
   - REST selectors with HTTP method + path.
   - RPC selectors with predicate-based matching.
5. Add or update tests near changed behavior.

## Commands

Use these standard commands from repo root:

```bash
make setup
make test
make lint
make vet
make tidy
```

For targeted checks:

```bash
cd <module> && go test ./...
```

`ronyup` module tests:

```bash
cd ronyup && go test ./...
```

## Guardrails

- Do not break `go.work` multi-module assumptions.
- Do not move code between `rony/` and `kit/` without explicit request.
- Keep naming and handler patterns consistent with surrounding code.
- Add comments only for non-obvious logic.
- Avoid unrelated formatting churn.

## When To Use ronyup MCP

Prefer `ronyup mcp` workflows when the user asks to scaffold or bootstrap:

- new workspace/service setup
- architecture-consistent API/service boilerplate
- generated patterns aligned with this repository conventions

## References

- Root overview: `README.MD`
- Architecture: `docs/architecture.md`
- Getting started: `docs/getting-started.md`
- ronyup + MCP usage: `docs/ronyup-guide.md`
