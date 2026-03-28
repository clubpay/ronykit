# RonyKIT

[![codecov](https://codecov.io/gh/clubpay/ronykit/branch/main/graph/badge.svg?token=6DALKCyYXv)](https://codecov.io/gh/clubpay/ronykit)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fclubpay%2Fronykit.svg?type=shield&issueType=license)](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit?ref=badge_shield&issueType=license)
[![Go Report Card](https://goreportcard.com/badge/github.com/clubpay/ronykit)](https://goreportcard.com/report/github.com/clubpay/ronykit)

A Go toolkit for building high-performance API servers. Define your contracts once,
get type-safe handlers, auto-generated docs, and client stubs — all with minimal boilerplate.

## Quick Start

### 1. Install the scaffolding tool

```bash
go install github.com/clubpay/ronykit/ronyup@latest
```

### 2. Create a new Project/Workspace

For interactive setup, use:

```bash
rony setup
```

Or specify all options directly:

```bash
ronyup setup workspace -r ./my-api -m github.com/you/my-api
```

### 3. Add a service

```bash
cd my-api
ronyup setup feature -p auth -n auth -t service
```

### 4. Run it

```bash
cd cmd/service && go run .
```

That's it — you have a running API server with a scaffolded project structure,
dependency injection, database migrations, and generated client stubs.

> **Using an AI-powered IDE?** Set up the [ronyup MCP server](./docs/ronyup-guide.md#mcp-server)
> to let your AI assistant scaffold and implement services for you.

## Hello World

Don't want scaffolding? Here's a complete API server in one file:

```go
package main

import (
    "context"
    "os"

    "github.com/clubpay/ronykit/rony"
)

type GreetRequest struct {
    Name string `json:"name"`
}

type GreetResponse struct {
    Message string `json:"message"`
}

func Greet(ctx *rony.SUnaryCtx, in GreetRequest) (*GreetResponse, error) {
    return &GreetResponse{Message: "Hello, " + in.Name + "!"}, nil
}

func main() {
    srv := rony.NewServer(
        rony.Listen(":8080"),
        rony.WithServerName("greeter"),
        rony.WithAPIDocs("/docs"),
        rony.UseSwaggerUI(),
    )

    rony.Setup(srv, "GreeterService", rony.EmptyState(),
        rony.WithUnary(Greet,
            rony.GET("/hello/{name}"),
            rony.POST("/hello"),
        ),
    )

    _ = srv.Run(context.Background(), os.Interrupt, os.Kill)
}
```

```bash
curl http://localhost:8080/hello/World
# {"message":"Hello, World!"}
```

Open http://localhost:8080/docs for auto-generated Swagger UI.

## Why RonyKit

- **Type-safe handlers** — request/response types are Go structs, validated at compile time
- **Auto-generated API docs** — Swagger UI / ReDoc / Scalar served directly from your handler types
- **Client stub generation** — generate Go and TypeScript clients from your service definitions
- **One handler, multiple protocols** — serve REST and RPC (WebSocket) from the same handler
- **Production-ready scaffolding** — `ronyup` generates projects with DI, repo layers, config, and migrations
- **AI-assisted development** — `ronyup mcp` integrates with Cursor, GoLand, and other MCP-capable IDEs
- **Minimal overhead** — built for performance with pluggable gateways (fasthttp, silverhttp)

## Documentation

| Guide | Description |
|-------|-------------|
| [Getting Started](./docs/getting-started.md) | Installation, first server, core concepts |
| [Cookbook](./docs/cookbook.md) | Authentication, pagination, validation, error handling, and more |
| [ronyup Guide](./docs/ronyup-guide.md) | Scaffolding CLI and MCP server for AI-assisted development |
| [Architecture](./docs/architecture.md) | How RonyKit works, project layout, component overview |
| [Advanced: Kit](./docs/advanced-kit.md) | Low-level toolkit for custom gateways and protocols |

## AI-Powered Development with ronyup MCP

`ronyup` includes a built-in [MCP server](https://modelcontextprotocol.io/) that lets
AI assistants in your IDE scaffold workspaces, plan services, and generate implementation
code — all following RonyKit best practices.

```json
{
  "mcpServers": {
    "ronyup": {
      "command": "ronyup",
      "args": ["mcp"]
    }
  }
}
```

Add this to `.cursor/mcp.json` (Cursor) or configure via Settings (GoLand).
See the full [ronyup MCP Guide](./docs/ronyup-guide.md#mcp-server) for details.

## Performance

RonyKIT is designed for high throughput with minimal framework overhead. See the
[TechEmpower JSON benchmark](https://www.techempower.com/benchmarks/#hw=ph&test=json&section=data-r22&l=zijocf-cn3)
for a reference point.

## Project Layout

```
rony/           Batteries-included framework (start here)
kit/            Low-level core (advanced users)
ronyup/         Scaffolding CLI + MCP server
std/gateways/   Gateway implementations (fasthttp, silverhttp, fastws, mcp)
std/clusters/   Cluster implementations (redis, p2p)
stub/           Client stub generation (Go, TypeScript)
flow/           Workflow helpers (Temporal integration)
x/              Extended utilities (di, telemetry, cache, i18n, apidoc, and more)
example/        Runnable examples
```

## Examples

Browse production-style patterns in the [Cookbook](./docs/cookbook.md) or explore the
[runnable examples](./example) directory.

## Build Commands

```bash
make setup    # Install tools (gotestsum, golangci-lint)
make test     # Run tests across all modules
make lint     # Lint all modules
make vet      # go vet all modules
make tidy     # go mod tidy all modules
```

## License

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fclubpay%2Fronykit.svg?type=large&issueType=license)](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit?ref=badge_large&issueType=license)
