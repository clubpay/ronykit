# RonyKIT

[![codecov](https://codecov.io/gh/clubpay/ronykit/branch/main/graph/badge.svg?token=6DALKCyYXv)](https://codecov.io/gh/clubpay/ronykit)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fclubpay%2Fronykit.svg?type=shield&issueType=license)](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit?ref=badge_shield&issueType=license)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fclubpay%2Fronykit.svg?type=shield&issueType=security)](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit?ref=badge_shield&issueType=security)
[![Go Report Card](https://goreportcard.com/badge/github.com/clubpay/ronykit)](https://goreportcard.com/report/github.com/clubpay/ronykit)

RonyKIT is a Go toolkit for high-performance API servers. Define service contracts one time. Use them to create type-safe handlers, API documentation, and client stubs.

## Quick start

### 1. Install the scaffolding tool

Install with Homebrew on macOS or Linux:

```bash
brew install clubpay/tap/ronyup
```

[Homebrew 6 and later](https://docs.brew.sh/Tap-Trust) requires explicit trust for third-party taps. The fully qualified name trusts the tap and installs `ronyup` in one step.

Upgrade `ronyup` with this command:

```bash
brew upgrade clubpay/tap/ronyup
```

You can also install with Go:

```bash
go install github.com/clubpay/ronykit/ronyup@latest
```

### 2. Create a project and workspace

Use interactive setup:

```bash
rony setup
```

Or specify all options:

```bash
ronyup setup workspace -r ./my-api -m github.com/you/my-api
```

### 3. Add a service

```bash
cd my-api
ronyup setup feature -p auth -n auth -t service
```

### 4. Run the server

```bash
cd cmd/all-in-one && go run .
```

The API server is now running. The project includes dependency injection, database migrations, and generated client stubs.

> **AI-powered IDE:** Configure the [ronyup MCP server](./docs/ronyup-guide.md#mcp-server). Your AI assistant can then scaffold and implement services.

## Hello World

This example shows a complete API server in one file. It does not use scaffolding.

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

Send a request to the server:

```bash
curl http://localhost:8080/hello/World
# {"message":"Hello, World!"}
```

Open http://localhost:8080/docs to view the generated Swagger UI.

## Why RonyKIT

- **Type-safe handlers:** Use Go structs for requests and responses. The Go compiler checks the types.
- **Generated API documentation:** Serve Swagger UI, ReDoc, or Scalar from handler types.
- **Client stub generation:** Generate Go and TypeScript clients from service definitions.
- **Multiple protocols:** Serve REST and RPC over WebSocket from the same handler.
- **Project scaffolding:** Generate projects with dependency injection, repository layers, configuration, and migrations.
- **AI-assisted development:** Use `ronyup mcp` with Cursor, GoLand, and other MCP-compatible IDEs.
- **Low overhead:** Use pluggable gateways, including fasthttp and silverhttp.

## Documentation

| Guide | Contents |
|-------|----------|
| [Getting Started](./docs/getting-started.md) | Installation, first server, and core concepts |
| [Cookbook](./docs/cookbook.md) | Authentication, pagination, validation, and error handling |
| [ronyup Guide](./docs/ronyup-guide.md) | Scaffolding CLI and MCP server |
| [Architecture](./docs/architecture.md) | Components, request flow, and project layout |
| [Advanced: Kit](./docs/advanced-kit.md) | Low-level tools for custom gateways and protocols |

## AI-assisted development with ronyup MCP

`ronyup` includes an [MCP server](https://modelcontextprotocol.io/). AI assistants can use this server to scaffold workspaces, plan services, and generate code. The tools follow RonyKIT conventions.

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

Add this configuration to `.cursor/mcp.json` for Cursor. For GoLand, use the MCP settings. See the [ronyup MCP guide](./docs/ronyup-guide.md#mcp-server) for instructions.

## Performance

RonyKIT is designed for high throughput and low framework overhead. See the [TechEmpower JSON benchmark](https://www.techempower.com/benchmarks/#hw=ph&test=json&section=data-r22&l=zijocf-cn3) for reference data.

## Project layout

```
rony/           High-level framework (start here)
kit/            Low-level core
ronyup/         Scaffolding CLI and MCP server
std/gateways/   Gateway implementations (fasthttp, silverhttp, fastws, mcp)
std/clusters/   Cluster implementations (redis, p2p)
stub/           Client stub generation (Go, TypeScript)
flow/           Workflow helpers (Temporal integration)
x/              Extended utilities (di, telemetry, cache, i18n, apidoc)
example/        Runnable examples
```

## Examples

See the [Cookbook](./docs/cookbook.md) for service patterns. See the [example directory](./example) for runnable services.

## Build commands

```bash
make setup    # Install tools (gotestsum, golangci-lint)
make test     # Run tests across all modules
make lint     # Lint all modules
make vet      # Run go vet across all modules
make tidy     # Run go mod tidy across all modules
```

## License

RonyKIT uses the [BSD 3-Clause License](LICENSE). [FOSSA](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit) tracks third-party licenses. See [License compliance](docs/compliance.md) for reports and scan issues.

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fclubpay%2Fronykit.svg?type=large&issueType=license)](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit?ref=badge_large&issueType=license)
