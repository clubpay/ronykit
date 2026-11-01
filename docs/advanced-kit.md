# Advanced: Use the kit package

The `kit` package is the low-level core of RonyKIT. It gives direct control of gateways, clusters, services, and contracts.

Most services must use `rony`. See the [Getting Started guide](./getting-started.md).

Use `kit` for these requirements:

- Custom gateways for non-HTTP protocols
- Custom cluster backends
- Direct control of the server pipeline
- Direct access to the EdgeServer API

## Quick start

Install the package:

```bash
go get github.com/clubpay/ronykit/kit@latest
```

### Minimal example

```go
package main

import (
	"context"
	"os"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

type EchoRequest struct {
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
}

type EchoResponse struct {
	ID string `json:"id"`
}

func echoHandler(ctx *kit.Context) {
	req := ctx.In().GetMsg().(*EchoRequest)
	ctx.In().
		Reply().
		SetMsg(&EchoResponse{ID: req.ID}).
		Send()
}

func EchoServiceDesc() *desc.Service {
	return desc.NewService("EchoService").
		SetEncoding(kit.JSON).
		AddContract(
			desc.NewContract().
				SetInput(&EchoRequest{}).
				SetOutput(&EchoResponse{}).
				AddRoute(desc.Route("GetEcho", fasthttp.GET("/echo/:id"))).
				AddRoute(desc.Route("PostEcho", fasthttp.POST("/echo"))).
				SetHandler(echoHandler),
		)
}

func main() {
	defer kit.NewServer(
		kit.WithGateway(
			fasthttp.MustNew(
				fasthttp.Listen(":80"),
			),
		),
		kit.WithServiceBuilder(EchoServiceDesc()),
	).
		Start(context.TODO()).
		PrintRoutes(os.Stdout).
		Shutdown(context.TODO(), os.Kill, os.Interrupt)
}
```

## Differences from rony

| Aspect | `rony` | `kit` |
|--------|--------|-------|
| Handler signature | `func(ctx, In) (*Out, error)` with typed messages | `func(ctx *kit.Context)` with raw messages |
| State management | Built-in reducer pattern | Manual state management |
| API documentation | Generated from types | Built manually with `kit/desc` |
| Gateway setup | Automatic fasthttp configuration | Manual gateway configuration |
| Server lifecycle | `rony.NewServer` and `rony.Setup` | `kit.NewServer` and `kit.WithServiceBuilder` |
| Route selectors | Helpers such as `rony.GET(path)` | `fasthttp.GET(path)` or custom selectors |
| Error handling | Return an error for framework serialization | Write the error to `ctx.Out()` |

## Components

### Handler

A raw handler uses `*kit.Context`:

```go
func myHandler(ctx *kit.Context) {
	req := ctx.In().GetMsg().(*MyRequest)

	ctx.In().
		Reply().
		SetMsg(&MyResponse{Data: req.Data}).
		Send()
}
```

### Service descriptor

Use `kit/desc` to build service descriptors:

```go
func MyServiceDesc() *desc.Service {
	return desc.NewService("MyService").
		SetEncoding(kit.JSON).
		AddContract(
			desc.NewContract().
				SetInput(&CreateRequest{}).
				SetOutput(&CreateResponse{}).
				AddRoute(desc.Route("Create", fasthttp.POST("/items"))).
				SetHandler(createHandler),
		).
		AddContract(
			desc.NewContract().
				SetInput(&GetRequest{}).
				SetOutput(&GetResponse{}).
				AddRoute(desc.Route("Get", fasthttp.GET("/items/:id"))).
				SetHandler(getHandler),
		)
}
```

### EdgeServer

Configure the gateways and services with `kit.NewServer`:

```go
srv := kit.NewServer(
	kit.WithGateway(
		fasthttp.MustNew(
			fasthttp.Listen(":8080"),
		),
	),
	kit.WithServiceBuilder(MyServiceDesc()),
	kit.WithServiceBuilder(AnotherServiceDesc()),
)
```

### Add a cluster

```go
import "github.com/clubpay/ronykit/std/clusters/rediscluster"

srv := kit.NewServer(
kit.WithGateway(fasthttp.MustNew(fasthttp.Listen(":8080"))),
kit.WithCluster(rediscluster.MustNew(
rediscluster.WithRedisAddr("localhost:6379"),
)),
kit.WithServiceBuilder(MyServiceDesc()),
)
```

## Storage layers

| Layer | Lifetime | Typical data |
|-------|----------|--------------|
| Context | One request | User ID and trace ID |
| Connection | One connection | WebSocket session data |
| Local | One server instance | In-memory caches and counters |
| Cluster | Multiple server instances | Shared state. A cluster bundle is required. |

## Custom gateways

Implement the `kit.Gateway` interface to create a custom gateway:

```go
type Gateway interface {
	Start(ctx context.Context, cfg GatewayStartConfig) error
	Shutdown(ctx context.Context) error
	Register(
		serviceName, contractID string,
		encoding kit.Encoding,
		sel kit.RouteSelector,
		input kit.Message,
	)
	Subscribe(d GatewayDelegate)
}
```

See `std/gateways/fasthttp` for an example implementation.

### Handler relay

A handler can forward an inbound HTTP or WebSocket request to a dynamic upstream. This can occur after authentication or session resolution.

Use `kit.RelayConn` and `kit.Relay`, `kit.RelayHTTP`, or `kit.RelayWebSocket`. In `rony` services, register the route with `rony.WithRelay` and use `RelayCtx`.

See MCP resource `architecture/handler-relay` and the [Cookbook](./cookbook.md).

## When to use kit

Use `kit` for these requirements:

- A non-HTTP protocol gateway, such as gRPC, MQTT, or custom TCP
- Direct control of the connection lifecycle
- A custom cluster backend
- Direct access that bypasses the `rony` abstractions

Use [`rony`](./getting-started.md) for other services.

## Next steps

- [Getting Started](./getting-started.md): Use the recommended `rony` framework.
- [Architecture](./architecture.md): Learn how components relate.
- [Runnable examples](../example): See `ex-01-rpc` and `ex-02-rest` for `kit` examples.
