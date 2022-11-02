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

# RonyKit

RonyKit is a library written in Go (Golang), provides the abstraction layer for creating an API server. By defining separate
components for each task, you are almost free to develop your API server using RPC (Proto/JSON/...), REST, GraphQL or even with your
in-house defined protocol. However, some components are implemented in the 'std' which make it easy to use RonyKit out of the box.
In RonyKIT, I wanted to make it more flexible but still very performant. I will maintain two bundles one based on
[fasthttp](https://github.com/valyala/fasthttp) and the other provides only rpc over websocket based on two very fast
libraries [gobwas](https://github.com/gobwas/ws) and [gnet](https://github.com/panjf2000/gnet), however you are not limited to use those,
and
can build your own Bundles. To have a better understanding of how it is going to work please check examples folder.
in [simple-rest-server](examples/simple-rest-server) it shows a very simple hello world app.
in [mixed-jsonrpc-rest](examples/mixed-jsonrpc-rest) it shows how you can write your services once but use them with multiple formats like
REST or RPC.

## Contents

- [RonyKIT](#ronykit)
	- [Contents](#contents)
	- [Installation](#installation)
	- [Quick start](#quick-start)
	- [Benchmark](#benchmark)
	- [Define Services and Contracts](#define-services-and-contracts)

## Installation

To install RonyKIT package, you need to install Go and set your Go workspace first.

First need to install[Go](https://golang.org/) (**version 1.17+ is required**), then you can use the below Go command to install
RonyKIT.

```sh
$ go get -u github.com/clubpay/ronykit/...
```

## Quick start

You can find some sample codes in the [Examples](./examples) folder.

## Define Services and Contracts

In RonyKIT you better to define the description of your services and their contracts using `desc` package,
and it provides you many helper methods. Although you can create your concrete implementations of kit.Service
and kit.Contract, but we strongly recommend you to use `desc` package.

### Define Service Descriptor

The following code demonstrates the steps you need to take to register your service.

```go
package main

import (
	"context"
	"syscall"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

type EchoRequest struct {
	RandomID int64 `json:"randomId"`
}

type EchoResponse struct {
	RandomID int64 `json:"randomId"`
}

var serviceDesc = desc.NewService("yourServiceUniqueName").
	AddContract(
		desc.NewContract().
			// Add the selector which this contract will be selected based of. Every bundle have its own selector
			// and here we use our std/fasthttp package.
			AddSelector(fasthttp.GET("/echo/:randomId")).
			// Define the input message, it defines the message your handler expects to receive.
			SetInput(&EchoRequest{}).
			// Define the output message, it defines the message your handler expects to return.
			SetOutput(&EchoResponse{}).
			// Set the handler which is going to handle this request.
			SetHandler(echoHandler),
	)

func echoHandler(ctx *kit.Context) {
	req, _ := ctx.In().GetMsg().(*EchoRequest)
	ctx.Out().SetMsg(
		&EchoResponse{
			RandomID: req.RandomID,
		},
	).Send()
}
func main() {
	app := kit.NewServer(
		desc.Register(serviceDesc),
	)

	// Start the server
	app.Start(context.Background())

	// Wait for signal to shut down
	app.Shutdown(context.Background(), syscall.SIGHUP)
}
```
