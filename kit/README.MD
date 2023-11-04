[//]: # (<a href="https://github.com/clubpay/ronykit/actions/workflows/build-and-test.yml?query=branch%3Amain">)

[//]: # (  <img alt="Build Status" src="https://img.shields.io/github/workflow/status/clubpay/ronykit/build-and-test/main?style=for-the-badge">)

[//]: # (</a>)
<a href="https://goreportcard.com/report/github.com/clubpay/ronykit">
  <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/clubpay/ronykit?style=for-the-badge">
</a>
<a href="https://github.com/clubpay/ronykit/releases">
  <img alt="GitHub release (latest by date including pre-releases)" src="https://img.shields.io/github/v/release/clubpay/ronykit?include_prereleases&style=for-the-badge">
</a>
<a href="https://codecov.io/gh/clubpay/ronykit/branch/main/">
    <img alt="Codecov Status" src="https://img.shields.io/codecov/c/github/clubpay/ronykit?style=for-the-badge">
</a>

## Contents

- [Kit](#kit)
	- [Contents](#contents)
	- [Installation](#installation)
	- [Quick start](#quick-start)
	- [Kit Components](#kit-components)
		- [1. Handler](#1-handler)
		- [2. Contract](#2-contract)
		- [3. Service](#3-service)
		- [4. Gateway](#4-gateway)
		- [5. Cluster](#5-cluster)
		- [6. EdgeServer](#6-edgeserver)
  - [KIT Storage layers](#kit-storage-layers)



# Kit

RonyKit is a library written in Go (Golang), provides the abstraction layer for creating an API server. By defining separate
components for each task, you are almost free to develop your API server using RPC (Proto/JSON/...), REST, GraphQL or even with your
in-house defined protocol. However, some components are implemented in the 'std' which make it easy to use RonyKit out of the box.
In RonyKIT, I wanted to make it more flexible but still very performant. I will maintain two bundles one based on
[fasthttp](https://github.com/valyala/fasthttp) and the other provides only rpc over websocket based on two very fast
libraries [gobwas](https://github.com/gobwas/ws) and [gnet](https://github.com/panjf2000/gnet), however you are not limited to use those,
and
can build your own Bundles. To have a better understanding of how it is going to work please check examples folder.
in [simple-rest-server](../example/ex-02-rest) it shows a very simple hello world app.
in [mixed-jsonrpc-rest](../example/ex-01-rpc) it shows how you can write your services once but use them with multiple formats like
REST or RPC.


## Installation

To install RonyKIT package, you need to install Go and set your Go workspace first.

First need to install[Go](https://golang.org/) (**version 1.17+ is required**), then you can use the below Go command to install
RonyKIT.

```sh
$ go get -u github.com/clubpay/ronykit/kit/...
```

## Quick start

You can find some sample codes in the [Examples](../example) folder.

## Define Services and Contracts

In RonyKIT(Kit) you better to define the description of your services and their contracts using `desc` package,
and it provides you many helper methods. Although you can create your concrete implementations of kit.Service
and kit.Contract, but we strongly recommend you to use `desc` package.

When you develop a handler with Kit package you need to think about the input and output of your API, and define
these DTOs by defining appropriate structs.

For example, if you want to implement an `Echo` handler you can write your handler like this:

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

Then you can write the service descriptor to define how your handler is going to be accessed from the Gateway of
your server. For example if you have an HTTP Gateway and you need your handler to be accessible using REST API,
you can define your service descriptor like this:

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
				SetInput(&EchoRequest{}).	// This tells that we expect the incoming request resolves to EchoRequest struct
				SetOutput(&EchoResponse{}). // This tells that we expect the outgoing response resolves to EchoResponse struct
				AddSelector(fasthttp.GET("/echo/:Id")). // This tells that we want to use echoHandler for GET requests with /echo/:Id path
				AddSelector(fasthttp.POST("/echo")). // This tells that we want to use echoHandler for POST requests with /echo path
				SetHandler(echoHandler), // This tells that we want to use echoHandler as the handler of this contract
		)
}
```

In the ServiceDescriptor we defined two REST endpoints which our Handler would be served.
RonyKIT's EdgeServer tries its best to fill the input struct (i.e., EchoRequest) from the parameters: urlParam, queryParam or request's
body.
If you have a more complex case, you can also add your own custom decoder when you add your selector to the contract.

The last step is to set up our EdgeServer and bind our desired Gateway bundle and the `MyServiceDesc`.
The following code shows how we can do that:

```go
package main

import (
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

---

# KIT Components

## 1. Handler

Handler is the function/method that accepts `kit.Context` as argument, and it will be called based on the
selector that is defined in the `desc.Service`.

## 2. Contract

Contract defines a business use-case that is very similar to what a function does, just it will be triggered
based on the selectors that we define the ServiceDescriptor.

## 3. Service

One or more Contracts that are doing some operations in a similar domain could be grouped in one Service. Each
EdgeServer can have one or more service attached.

## 4. Gateway

Gateways handle the endpoints of the service. For example, if you need to provide REST apis you can use one of the
standard gateway bundles : fasthttp or silverhttp
For advanced cases you can develop your own Gateway, but it is not necessary for most cases.

## 5. Cluster

Cluster defines the relation between different instances of the EdgeServer. This bundle is optional but if attached
to EdgeServer then in multi instance cases it can help and provides some facilities to share data between different
instances of your EdgeServer.

## 6. EdgeServer

EdgeServer is the main component of the RonyKIT which glues different components together to make it a working
server.

---

---

---

# KIT Storage layers

When we develop API server usually we need to store data with different lifecycles. Sometimes we need to store data
that lives between different handlers, in some cases, we need to keep the data between different requests of a connection.
(i.e. for websocket connection we want to keep some data for next request from this connection).
Basically in RonyKIT we provide 4 layers of storage:

| Layer      | Lifecycle                                                                                                                                                                        |
|------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Context    | in the request, but available in all the handlers                                                                                                                                |
| Connection | in the connection, for REST apis usually the lifecycle of Connection storage and Context are same, but for websocket data will be available until websocket connection is active |
| Local      | this is shared between different contracts, services and is in the local memory of the server                                                                                    |
| Cluster    | this is shared between different instances of the EdgeServer. This is enabled ONLY if a Cluster bundle is attached                                                               |

---


