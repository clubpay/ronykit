# Architecture

This document describes RonyKIT components, request flow, and repository layout.

## Two layers

RonyKIT has two abstraction levels:

| Level | Package | Purpose |
|-------|---------|---------|
| High-level | `rony` | Typed handlers, generated documentation, and state management. Start with this package. |
| Low-level | `kit` | Core components for custom gateways, protocols, and direct control. |

`rony` uses `kit`. A `rony` handler can access the `kit` API with `ctx.KitCtx()`.

## Request flow

```
Client
  │
  ▼
Gateway (fasthttp / silverhttp / fastws)
  │
  ▼
EdgeServer
  │
  ▼
Service lookup
  │
  ▼
Contract match (route selector)
  │
  ▼
Middleware chain
  │
  ▼
Handler
  │
  ▼
Response
```

1. A gateway receives an inbound HTTP or WebSocket request.
2. The EdgeServer selects the service for the request.
3. A route selector selects the contract in the service.
4. Service middleware runs, then contract middleware runs.
5. The handler processes the request and returns a response.

## Key abstractions

### EdgeServer

The EdgeServer connects gateways, clusters, and services. In the `rony` layer, `rony.Server` wraps an EdgeServer and supplies default configuration.

### Gateway

A gateway receives inbound traffic. RonyKIT includes these gateway implementations:

| Gateway | Package | Implementation |
|---------|---------|----------------|
| fasthttp | `std/gateways/fasthttp` | HTTP gateway based on [valyala/fasthttp](https://github.com/valyala/fasthttp) |
| silverhttp | `std/gateways/silverhttp` | HTTP gateway based on [silverlining](https://github.com/go-www/silverlining) |
| fastws | `std/gateways/fastws` | WebSocket gateway based on [gnet](https://github.com/panjf2000/gnet) and [gobwas/ws](https://github.com/gobwas/ws) |
| mcp | `std/gateways/mcp` | Model Context Protocol gateway |

`rony.NewServer()` configures the fasthttp gateway automatically.

### Cluster

A cluster is optional. It coordinates multiple EdgeServer instances and shares state between them.

| Cluster | Package | Implementation |
|---------|---------|----------------|
| rediscluster | `std/clusters/rediscluster` | Cluster based on Redis |
| p2pcluster | `std/clusters/p2pcluster` | Peer-to-peer cluster based on [libp2p](https://github.com/libp2p/go-libp2p) |

### Service

A service groups related contracts. One EdgeServer can host multiple services. Use `rony.Setup()` to register services.

### Contract

A contract defines one API operation. It contains input types, output types, route selectors, and a handler.

In `rony`, use `rony.WithUnary()`, `rony.WithStream()`, or `rony.WithRelay()` to create contracts. Use `rony.WithRelay()` for HTTP and WebSocket pass-through proxy routes.

### Context

The context stores request data in four layers:

| Layer | Lifetime | Access |
|-------|----------|--------|
| Context | One request | Available in all handlers |
| Connection | One connection | Kept across requests on the same WebSocket connection |
| Local | One server instance | Shared by all contracts and services on the instance |
| Cluster | Multiple server instances | Shared by EdgeServer instances. A cluster bundle is required. |

## Repository layout

```
ronykit/
├── rony/              High-level framework (start here)
│   ├── server.go      Server creation and lifecycle
│   ├── setup.go       Service registration with contracts
│   ├── ctx.go         Type-safe handler contexts
│   ├── selector.go    Route helpers (GET, POST, etc.)
│   └── errs/          Structured error handling
│
├── kit/               Low-level core
│   ├── edge.go        EdgeServer implementation
│   ├── ctx.go         Raw request context
│   └── desc/          Service/Contract descriptors
│
├── ronyup/            Scaffolding CLI + MCP server
│   ├── cmd/           CLI commands (setup, text, mcp, template)
│   └── internal/      Skeleton templates, MCP tools
│
├── std/
│   ├── gateways/      Gateway implementations
│   │   ├── fasthttp/
│   │   ├── silverhttp/
│   │   ├── fastws/
│   │   └── mcp/
│   └── clusters/      Cluster implementations
│       ├── rediscluster/
│       └── p2pcluster/
│
├── stub/              Client stub generation (Go, TypeScript)
├── flow/              Workflow helpers (Temporal integration)
├── testenv/           Testing environment utilities
│
├── x/                 Extended utilities
│   ├── di/            Dependency injection (uber/fx helpers)
│   ├── telemetry/     Observability (logging, tracing, metrics)
│   ├── apidoc/        OpenAPI doc generation
│   ├── cache/         Caching utilities
│   ├── datasource/    Database and Redis connection helpers
│   ├── i18n/          Internationalization
│   ├── ratelimit/     Rate limiting
│   ├── settings/      Configuration management (Viper-backed)
│   ├── batch/         Batch processing
│   ├── rkit/          Common helpers
│   ├── testkit/       Testing utilities
│   └── p/             Additional primitives
│
└── example/           Runnable examples
    ├── ex-01-rpc/
    ├── ex-02-rest/
    ├── ex-04-stubgen/
    ├── ex-05-counter/
    └── ...
```

## Encoding

RonyKIT supports these encoding formats:

| Format | Purpose |
|--------|---------|
| JSON | Default format. Uses struct `json` tags. |
| Protobuf | Protocol Buffers encoding |
| MessagePack | Binary encoding |
| Multipart | `multipart/form-data` for file uploads |
| Custom | Encoding supplied by a custom codec |

## Routing

RonyKIT has two routing strategies:

- **REST selectors:** Match an HTTP method and path pattern. Example: `GET /users/{id}`.
- **RPC selectors:** Match a predicate over WebSocket. Example: `RPC("getUser")`.

One handler can serve REST and RPC routes. Define the handler one time and register both route types.

## Extended utilities

The `x/` directory contains optional packages for RonyKIT services. Use these packages with `ronyup` scaffolds when they meet the service requirements.

| Package | Purpose |
|---------|---------|
| `x/di` | Dependency injection helpers for uber/fx |
| `x/settings` | Configuration management with Viper |
| `x/telemetry/logkit` | Structured logging |
| `x/telemetry/tracekit` | Distributed tracing with OpenTelemetry |
| `x/telemetry/meterkit` | Metrics collection |
| `x/apidoc` | OpenAPI document generation |
| `x/cache` | Caching utilities |
| `x/datasource` | Database and Redis connection management |
| `x/i18n` | Internationalization support |
| `x/ratelimit` | Rate limiting |
| `x/batch` | Batch processing |
| `x/rkit` | Common helper functions |
| `x/testkit` | Testing utilities |

## Next steps

- [Getting Started](./getting-started.md): Build a server.
- [Cookbook](./cookbook.md): Use service patterns and examples.
- [ronyup Guide](./ronyup-guide.md): Use the scaffolding CLI and MCP server.
- [Advanced: Kit](./advanced-kit.md): Build custom gateways with low-level tools.
