# Kit

[![Go Reference](https://pkg.go.dev/badge/github.com/clubpay/ronykit/kit.svg)](https://pkg.go.dev/github.com/clubpay/ronykit/kit) [![Go Report Card](https://goreportcard.com/badge/github.com/clubpay/ronykit/kit)](https://goreportcard.com/report/github.com/clubpay/ronykit/kit)

Kit is the low-level toolkit in RonyKIT. It gives direct control of gateways, clusters, services, and contracts with low framework overhead.

Use Kit for custom protocols, custom gateway or cluster implementations, and direct control of the server pipeline.

- [Installation](#installation)
- [Quick start](#quick-start)
- [Kit components](#kit-components)
- [Kit storage layers](#kit-storage-layers)
- [Kit standard implementations](#kit-standard-implementations)

## Installation

Install Go 1.17 or later. Then install Kit:

```sh
go get -u github.com/clubpay/ronykit/kit/...
```

## Quick start

Follow these steps:

1. Define DTOs for input and output.
2. Implement a handler that reads the input message and writes a response.
3. Describe the service with `kit/desc` and bind routes.
4. Start an EdgeServer with a gateway bundle.

Minimal example:

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

func MyServiceDesc() *desc.Service {
	return desc.NewService("MyServiceName").
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
		kit.WithServiceBuilder(MyServiceDesc()),
	).
		Start(context.TODO()).
		PrintRoutes(os.Stdout).
		Shutdown(context.TODO(), os.Kill, os.Interrupt)
}
```

See [example](../example) for more samples.

## Kit components

### 1. Handler

A handler is a function or method that accepts `kit.Context`. A contract route invokes the handler.

### 2. Contract

A contract defines a business use case. Gateway selectors expose this use case.

### 3. Service

A service groups contracts from the same domain. Each EdgeServer can host multiple services.

### 4. Gateway

Gateways receive inbound traffic. Use standard gateway bundles such as `fasthttp` or `silverhttp` for REST APIs. Implement a custom gateway for other requirements.

### 5. Cluster

A cluster defines how EdgeServer instances share data. This bundle is optional. Use it only for cross-instance state.

### 6. EdgeServer

The EdgeServer connects the RonyKIT components into a running server.

## Kit storage layers

RonyKIT provides four storage layers for different data lifetimes:

| Layer | Lifetime |
|-------|----------|
| Context | One request. Available in all handlers. |
| Connection | One connection. Keeps WebSocket data while the connection stays open. |
| Local | One server instance. Shares memory between contracts and services. |
| Cluster | Multiple EdgeServer instances. Requires a cluster bundle. |

## Handler relay

`kit.RelayConn` lets a handler forward an inbound HTTP or WebSocket request to a dynamic upstream URL. Authentication or session logic can run before the relay.

Use `kit.Relay`, `kit.RelayHTTP`, `kit.RelayWebSocket`, and `kit.RelayConfig`. `std/gateways/fasthttp` connections implement these operations.

In application services, use `rony.WithRelay` and `RelayCtx`. See `docs/cookbook.md` and ronyup MCP resource `architecture/handler-relay`.

Use `rony.WithReverseProxy` for static gateway proxy paths.

## Kit standard implementations

The repository includes standard gateway and cluster bundles. Select the gateway and cluster for your requirements. You can also use the high-level [rony](../rony/README.MD) package.

| Package | Bundle type | Implementation |
|---------|-------------|----------------|
| fasthttp | Gateway | Gateway based on [fasthttp](https://github.com/valyala/fasthttp) |
| fastws | Gateway | Gateway based on [gnet](https://github.com/panjf2000/gnet) and [gobwas](https://github.com/gobwas/ws) |
| silverhttp | Gateway | Gateway based on the [silverlining](https://github.com/go-www/silverlining) HTTP server |
| rediscluster | Cluster | Cluster based on [Redis](https://github.com/go-redis/redis) |
| p2pcluster | Cluster | Cluster based on [libp2p](https://github.com/libp2p/go-libp2p) |
