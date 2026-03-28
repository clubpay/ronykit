# Advanced: Using the Kit Package

The `kit` package is the low-level core of RonyKIT. It gives you granular control
over gateways, clusters, services, and contracts. **Most users should use `rony`
instead** — see the [Getting Started guide](./getting-started.md).

Use `kit` when you need:

- Custom gateway implementations (non-HTTP protocols)
- Custom cluster backends
- Manual control over the server pipeline
- Direct access to the EdgeServer API

---

## Quick Start

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

---

## Key Differences from `rony`

| Aspect                | `rony`                                | `kit`                                      |
|-----------------------|---------------------------------------|--------------------------------------------|
| **Handler signature** | `func(ctx, In) (*Out, error)` — typed | `func(ctx *kit.Context)` — raw             |
| **State management**  | Built-in reducer pattern              | Manual                                     |
| **API docs**          | Auto-generated from types             | Build with `kit/desc` manually             |
| **Gateway setup**     | Automatic (fasthttp by default)       | Manual — you wire gateways yourself        |
| **Server lifecycle**  | `rony.NewServer` + `rony.Setup`       | `kit.NewServer` + `kit.WithServiceBuilder` |
| **Route selectors**   | `rony.GET(path)` helpers              | `fasthttp.GET(path)` or custom selectors   |
| **Error handling**    | Return error, framework serializes    | Manual: write to `ctx.Out()` yourself      |

---

## Components

### Handler

A raw handler operates on `*kit.Context`:

```go
func myHandler(ctx *kit.Context) {
req := ctx.In().GetMsg().(*MyRequest)

ctx.In().
Reply().
SetMsg(&MyResponse{Data: req.Data}).
Send()
}
```

### Service Descriptor

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

Wire it all together:

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

### Adding a Cluster

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

---

## Storage Layers

| Layer          | Lifecycle           | Use Case                                |
|----------------|---------------------|-----------------------------------------|
| **Context**    | Per request         | Request-scoped data (user ID, trace ID) |
| **Connection** | Per connection      | WebSocket session data                  |
| **Local**      | Per server instance | In-memory caches, counters              |
| **Cluster**    | Cross-instance      | Shared state (requires cluster bundle)  |

---

## Custom Gateways

To implement a custom gateway, implement the `kit.Gateway` interface:

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

See `std/gateways/fasthttp` for a reference implementation.

---

## When to Use Kit

- You're building a non-HTTP protocol gateway (gRPC, MQTT, custom TCP)
- You need fine-grained control over connection lifecycle
- You're implementing a custom cluster backend
- You need to bypass the `rony` abstractions for performance reasons

For everything else, use [`rony`](./getting-started.md).

---

## Next Steps

- [Getting Started](./getting-started.md) — the recommended `rony` approach
- [Architecture](./architecture.md) — how all components fit together
- [Runnable examples](../example) — see `ex-01-rpc` and `ex-02-rest` for kit usage
