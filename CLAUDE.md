# CLAUDE.md

## Project Overview

RonyKit is a Go toolkit for building high-performance network services. It is organized
as a **Go workspace** (`go.work`, Go 1.25+) containing 27+ independent modules.

Two abstraction levels exist:

- **`kit/`** -- low-level core (EdgeServer, Gateway, Cluster, Contract, Context)
- **`rony/`** -- high-level, batteries-included framework built on `kit`

## Repository Layout

```
kit/            Core building blocks (EdgeServer, contracts, context, codecs)
rony/           High-level framework (server, typed context, state management)
flow/           Workflow helpers (Temporal SDK integration)
stub/           Client stub generation (Go / TypeScript)
ronyup/         Project scaffolding CLI
testenv/        Testing environment utilities
std/
  gateways/     Gateway implementations (fasthttp, silverhttp, fastws, mcp)
  clusters/     Cluster implementations (rediscluster, p2pcluster)
x/              Extended utilities (di, telemetry, apidoc, cache, datasource,
                i18n, ratelimit, batch, settings, testkit, rkit, p)
example/        Runnable examples (ex-01 through ex-11)
scripts/        Build & maintenance scripts
docs/           Diagrams and extra documentation
```

## Build & Development Commands

```sh
make setup       # Install tools (gotestsum, golangci-lint)
make test        # Run tests across all modules (excludes example/, ronyup/)
make lint        # Lint all modules (excludes example/)
make vet         # go vet all modules (excludes example/)
make tidy        # go mod tidy all modules (excludes example/)
```

To test a single module: `cd <module> && go test ./...`
To test ronyup specifically: `cd ronyup && go test ./...`

## Testing

- Framework: **Ginkgo v2** with **Gomega** matchers.
- Test runner: `gotestsum` with `--format pkgname-and-test-fails`.
- Coverage: `covermode=atomic`, generates `coverage.out` per module.
- Run focused module tests first; use `make test` for broader validation.

## Architecture Quick Reference

**Request flow:**
```
Client -> Gateway -> northBridge -> EdgeServer -> Contract lookup -> Handler chain -> Response
```

**Key abstractions:**
- **EdgeServer** -- orchestrator that binds Gateways, Clusters, and Services.
- **Gateway** -- inbound traffic (fasthttp, silverhttp, fastws, mcp).
- **Cluster** -- multi-instance coordination (Redis, libp2p P2P).
- **Service** -- logical grouping of Contracts.
- **Contract** -- single API operation (input/output types, route selectors, handlers).
- **Context** -- request-scoped state with four storage layers:
  per-request, per-connection, per-service (local), cluster-wide.

**Encoding:** JSON, Protobuf, MessagePack, multipart/form-data, or custom.

**Routing:** `RESTRouteSelector` (HTTP method + path) and `RPCRouteSelector` (predicate).

## Code Conventions

- Preserve existing architecture and naming conventions.
- Add comments only when logic is non-obvious.
- Do not introduce unrelated formatting churn.
- Keep edits scoped to the specific task.
- Read the relevant package/module README before changing behavior.
- Each module under `std/` and `x/` has its own `go.mod`; respect module boundaries.

## Git Practices

- Keep commits atomic with clear intent.
- Do not revert unrelated local changes.
- Avoid destructive git operations unless explicitly requested.
- Main branch: `main`.
