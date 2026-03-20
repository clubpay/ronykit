# Architecture

This document explains how RonyKIT works, how the components fit together, and
how the repository is organized.

## Two Layers

RonyKIT provides two levels of abstraction:

| Layer | Package | Description |
|-------|---------|-------------|
| **High-level** | `rony` | Batteries-included framework. Type-safe handlers, built-in docs, state management. **Start here.** |
| **Low-level** | `kit` | Core building blocks. Use when you need custom gateways, protocols, or deeper control. |

`rony` is built on top of `kit`. You can always drop down to `kit` APIs from within
`rony` handlers using `ctx.KitCtx()`.

---

## Request Flow

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

1. A **Gateway** receives inbound traffic (HTTP, WebSocket, etc.).
2. The **EdgeServer** routes the request to the matching **Service**.
3. Within the service, the **Contract** is matched by its route selector.
4. The **Middleware chain** runs (service-level, then contract-level).
5. The **Handler** processes the request and returns a response.

---

## Key Abstractions

### EdgeServer

The main orchestrator. Binds gateways, clusters, and services together.
In the `rony` layer, `rony.Server` wraps an EdgeServer with opinionated defaults.

### Gateway

Handles inbound traffic. RonyKIT ships with several gateway implementations:

| Gateway | Package | Description |
|---------|---------|-------------|
| fasthttp | `std/gateways/fasthttp` | High-performance HTTP gateway using [valyala/fasthttp](https://github.com/valyala/fasthttp) |
| silverhttp | `std/gateways/silverhttp` | HTTP gateway using [silverlining](https://github.com/go-www/silverlining) |
| fastws | `std/gateways/fastws` | WebSocket gateway using [gnet](https://github.com/panjf2000/gnet) + [gobwas/ws](https://github.com/gobwas/ws) |
| mcp | `std/gateways/mcp` | Model Context Protocol gateway |

When using `rony.NewServer()`, the fasthttp gateway is configured automatically.

### Cluster

Optional. Enables multi-instance coordination for shared state across EdgeServer
instances.

| Cluster | Package | Description |
|---------|---------|-------------|
| rediscluster | `std/clusters/rediscluster` | Redis-backed cluster |
| p2pcluster | `std/clusters/p2pcluster` | Peer-to-peer cluster using [libp2p](https://github.com/libp2p/go-libp2p) |

### Service

A logical grouping of contracts. One EdgeServer can host multiple services.
Services are registered with `rony.Setup()`.

### Contract

A single API operation. Defines the input/output types, route selectors, and handler.
In the `rony` layer, contracts are created implicitly with `rony.WithUnary()` and
`rony.WithStream()`.

### Context

Request-scoped state with four storage layers:

| Layer | Lifecycle | Access |
|-------|-----------|--------|
| **Context** | Per request | Available in all handlers |
| **Connection** | Per connection | Persists across requests on the same connection (WebSocket) |
| **Local** | Per server instance | Shared between all contracts and services |
| **Cluster** | Cross-instance | Shared across EdgeServer instances (requires a Cluster bundle) |

---

## Repository Layout

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

---

## Encoding

RonyKIT supports multiple encoding formats:

| Format | Description |
|--------|-------------|
| JSON | Default. Uses struct `json` tags |
| Protobuf | Protocol Buffers encoding |
| MessagePack | Binary encoding |
| Multipart | `multipart/form-data` for file uploads |
| Custom | Implement your own codec |

---

## Routing

Two routing strategies are available:

- **REST selectors** — HTTP method + path pattern (e.g., `GET /users/{id}`)
- **RPC selectors** — Predicate-based routing over WebSocket (e.g., `RPC("getUser")`)

A single handler can serve both REST and RPC routes, which is how RonyKIT achieves
"define once, serve everywhere."

---

## Extended Utilities (`x/`)

The `x/` directory contains optional packages that integrate with the RonyKIT
ecosystem. These are recommended when using the `ronyup` scaffolding:

| Package | Purpose |
|---------|---------|
| `x/di` | Dependency injection helpers for uber/fx |
| `x/settings` | Configuration management with Viper |
| `x/telemetry/logkit` | Structured logging |
| `x/telemetry/tracekit` | Distributed tracing (OpenTelemetry) |
| `x/telemetry/meterkit` | Metrics collection |
| `x/apidoc` | OpenAPI document generation |
| `x/cache` | Caching utilities |
| `x/datasource` | Database and Redis connection management |
| `x/i18n` | Internationalization support |
| `x/ratelimit` | Rate limiting |
| `x/batch` | Batch processing |
| `x/rkit` | Common helper functions |
| `x/testkit` | Testing utilities |

---

## Next Steps

- [Getting Started](./getting-started.md) — build your first server
- [Cookbook](./cookbook.md) — production patterns and examples
- [ronyup Guide](./ronyup-guide.md) — scaffolding and MCP server
- [Advanced: Kit](./advanced-kit.md) — low-level toolkit for custom gateways
