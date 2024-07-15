# Kit
[![Go Reference](https://pkg.go.dev/badge/github.com/clubpay/ronykit/kit.svg)](https://pkg.go.dev/github.com/clubpay/ronykit/kit)
[![Go Report Card](https://goreportcard.com/badge/github.com/clubpay/ronykit/kit)](https://goreportcard.com/report/github.com/clubpay/ronykit/kit)


RonyKit is a Go (Golang) library that provides an abstraction layer for creating API servers. By defining separate components
for each task, you can develop your API server using RPC (Proto/JSON/...), REST, GraphQL, or even custom protocols.
Some components are implemented in the 'std' directory, making it easy to use RonyKit out of the box. RonyKit aims to be
both flexible and performant. It includes two bundles: one based on [fasthttp](https://github.com/valyala/fasthttp) and
another providing RPC over WebSocket using [gobwas](https://github.com/gobwas/ws) and [gnet](https://github.com/panjf2000/gnet).
However, you are free to build your own bundles. For examples, check the examples folder, such
as [simple-rest-server](../example/ex-02-rest) for a simple hello world app and [mixed-jsonrpc-rest](../example/ex-01-rpc) for services
using multiple formats like REST and RPC.

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Kit Components](#kit-components)
	- [1. Handler](#1-handler)
	- [2. Contract](#2-contract)
	- [3. Service](#3-service)
	- [4. Gateway](#4-gateway)
	- [5. Cluster](#5-cluster)
	- [6. EdgeServer](#6-edgeserver)
- [KIT Storage Layers](#kit-storage-layers)

## Installation

To install the RonyKIT package, you need to first install Go and set your Go workspace.

First, install [Go](https://golang.org/) (version 1.17+ is required). Then use the following command to install RonyKIT:

```sh
$ go get -u github.com/clubpay/ronykit/kit/...
```

## Quick Start

You can find sample code in the [Examples](../example) folder.

## Define Services and Contracts

In RonyKIT (Kit), it's best to define the description of your services and their contracts using the `desc` package, which
provides many helper methods. While you can create your own concrete implementations of `kit.Service` and `kit.Contract`,
we strongly recommend using the `desc` package.

When developing a handler with the Kit package, you need to consider the input and output of your API and define these
DTOs by creating appropriate structs.

For example, to implement an `Echo` handler, you can write your handler like this:

```go
package main

import "github.com/clubpay/ronykit/kit"

type EchoRequest struct {
    ID string `json:"Id"`
    Timestamp int64 `json:"timestamp"`
}

type EchoResponse struct {
    ID string `json:"Id"`
}

func echoHandler(ctx *kit.Context) {
    req := ctx.In().GetMsg().(*EchoRequest)

    ctx.In().
        Reply().
        SetMsg(&EchoResponse{ID: req.ID}).
        Send()
}
```

Then you can write the service descriptor to define how your handler will be accessed from the Gateway of your server.
For an HTTP Gateway to access your handler using a REST API, define your service descriptor like this:

```go
package main

import (
    "github.com/clubpay/ronykit/kit"
    "github.com/clubpay/ronykit/kit/desc"
    "github.com/clubpay/ronykit/std/gateways/fasthttp"
)

func MyServiceDesc() *desc.Service {
    return desc.NewService("MyServiceName").
        SetEncoding(kit.JSON).
        AddContract(
            desc.NewContract().
                SetInput(&EchoRequest{}).   // Incoming request resolves to EchoRequest struct
                SetOutput(&EchoResponse{}). // Outgoing response resolves to EchoResponse struct
                AddSelector(fasthttp.GET("/echo/:Id")). // Use echoHandler for GET requests with /echo/:Id path
                AddSelector(fasthttp.POST("/echo")).    // Use echoHandler for POST requests with /echo path
                SetHandler(echoHandler),                // Use echoHandler as the handler for this contract
        )
}
```

In the `ServiceDescriptor`, we defined two REST endpoints which our Handler would serve. RonyKIT's EdgeServer tries its best to fill the
input struct (i.e., EchoRequest) from URL parameters, query parameters, or the request's body. For more complex cases, you can add
your own custom decoder when adding your selector to the contract.

The final step is to set up the EdgeServer, bind the desired Gateway bundle, and the `MyServiceDesc`. The following code shows
how to do this:

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
    ID string `json:"Id"`
    Timestamp int64 `json:"timestamp"`
}

type EchoResponse struct {
    ID string `json:"Id"`
}

func echoHandler(ctx *kit.Context) {
    req := ctx.In().GetMsg().(*EchoRequest)

    ctx.In().
        Reply().
        SetMsg(&EchoResponse{ID: req.ID}).
        Send()
}

var MyServiceDesc desc.ServiceDescFunc = func() *desc.Service {
    return desc.NewService("MyServiceName").
        SetEncoding(kit.JSON).
        AddContract(
            desc.NewContract().
                SetInput(&EchoRequest{}).
                SetOutput(&EchoResponse{}).
                AddSelector(fasthttp.GET("/echo/:Id")).
                AddSelector(fasthttp.POST("/echo")).
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
        kit.WithServiceDesc(MyServiceDesc()),
    ).
        Start(context.TODO()).
        PrintRoutes(os.Stdout).
        Shutdown(context.TODO(), os.Kill, os.Interrupt)
}
```

# KIT Components

## 1. Handler

A Handler is a function/method that accepts `kit.Context` as an argument. It is called based on the
selector defined in the `desc.Service`.

## 2. Contract

A Contract defines a business use case similar to a function, triggered based on the selectors defined
in the `ServiceDescriptor`.

## 3. Service

One or more Contracts performing operations in a similar domain can be grouped into one Service. Each
EdgeServer can have multiple services attached.

## 4. Gateway

Gateways handle the service endpoints. For REST APIs, you can use standard gateway bundles
like `fasthttp` or `silverhttp`. For advanced cases, you can develop your own Gateway, but it is
unnecessary for most cases.

## 5. Cluster

A Cluster defines the relationship between different instances of the EdgeServer. This bundle is optional but,
if attached to the EdgeServer, it provides facilities to share data between different instances
of your EdgeServer.

## 6. EdgeServer

EdgeServer is the main component of RonyKIT, gluing different components together to create a working server.

# KIT Storage Layers

When developing an API server, you often need to store data with different lifecycles.
RonyKIT provides four storage layers:

| Layer      | Lifecycle                                                                                                              |
|------------|------------------------------------------------------------------------------------------------------------------------|
| Context    | Per request, available in all handlers                                                                                 |
| Connection | Per connection, useful for WebSocket data that persists until the connection is active                                 |
| Local      | Shared between different contracts and services, stored in the server's local memory                                   |
| Cluster    | Shared between different instances of the EdgeServer, enabled ONLY if a Cluster bundle is attached                     |ß
