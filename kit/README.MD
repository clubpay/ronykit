# Kit

[![Go Reference](https://pkg.go.dev/badge/github.com/clubpay/ronykit/kit.svg)](https://pkg.go.dev/github.com/clubpay/ronykit/kit)
[![Go Report Card](https://goreportcard.com/badge/github.com/clubpay/ronykit/kit)](https://goreportcard.com/report/github.com/clubpay/ronykit/kit)

Kit is the low-level toolkit that powers RonyKIT. It gives you granular control over gateways, clusters,
services, and contracts, while keeping the framework overhead minimal. Use Kit when you need custom
protocols, bespoke gateway/cluster choices, or deeper control over the server pipeline.

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Kit Components](#kit-components)
- [KIT Storage Layers](#kit-storage-layers)
- [KIT Standard Implementations](#kit-standard-implementations)

## Installation

Install Go (version 1.17+ is required), then:

```sh
go get -u github.com/clubpay/ronykit/kit/...
```

## Quick Start

The recommended workflow is:

1. Define DTOs for input and output.
2. Implement a handler that reads the input message and writes a response.
3. Describe the service using `kit/desc` and bind routes.
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

For more samples, see [example](../example).

## Kit Components

## 1. Handler

A Handler is a function or method that accepts `kit.Context` and is invoked based on a contract route.

## 2. Contract

A Contract defines a business use case and how it is exposed via gateway selectors.

## 3. Service

One or more Contracts performing operations in a similar domain can be grouped into one Service.
Each EdgeServer can have multiple services attached.

## 4. Gateway

Gateways handle inbound traffic. For REST APIs, use standard gateway bundles like `fasthttp` or
`silverhttp`. For advanced cases, you can develop your own gateway.

## 5. Cluster

A Cluster defines how multiple EdgeServer instances share data. This bundle is optional and only needed
when you require cross-instance state.

## 6. EdgeServer

EdgeServer is the main component of RonyKIT, gluing different components together to create a working server.

## KIT Storage Layers

When developing an API server, you often need to store data with different lifecycles.
RonyKIT provides four storage layers:

| Layer      | Lifecycle                                                                 |
|------------|---------------------------------------------------------------------------|
| Context    | Per request, available in all handlers                                    |
| Connection | Per connection, useful for WebSocket data that persists while connected   |
| Local      | Shared between contracts and services, stored in the server's memory      |
| Cluster    | Shared between EdgeServer instances, requires a Cluster bundle            |

## KIT Standard Implementations

This repository contains standard gateway and cluster bundles for Kit. You can mix and match the
Gateway and Cluster that suit your needs, or use the batteries-included [rony](../rony/README.MD) package.

| Package      | Bundle Type | Description                                                                                                                          |
|--------------|-------------|--------------------------------------------------------------------------------------------------------------------------------------|
| fasthttp     | Gateway     | The Gateway bundle implemented using the [fasthttp](https://github.com/valyala/fasthttp) framework                                   |
| fastws       | Gateway     | The Gateway bundle implemented using [gnet](https://github.com/panjf2000/gnet) and [gobwas](https://github.com/gobwas/ws) frameworks |
| silverhttp   | Gateway     | The Gateway bundle implemented using the [silverlining](https://github.com/go-www/silverlining) HTTP server                          |
| rediscluster | Cluster     | The Cluster bundle implemented using [redis](https://github.com/go-redis/redis)                                                      |
| p2pcluster   | Cluster     | The Cluster bundle implemented using [libp2p](https://github.com/libp2p/go-libp2p)                                                    |
