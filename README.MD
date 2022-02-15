# RonyKit

RonyKit is a library written in Go (Golang), provides the abstraction layer for creating a cluster aware API server. By defining separate
components for each task, you are almost free to develop your API server using RPC (Proto/JSON/...), REST, GraphQL or even with your
in-house defined protocol. However, some components are implemented in the 'std' which make it easy to use RonyKit out of the box. RonyKIT
is a successor of [Rony](https://github.com/ronaksoft/rony), with lessons from using it in multiple productions. In RonyKIT, I wanted to
make it more flexible but still very performant. I will maintain two bundles one based on [fasthttp](https://github.com/valyala/fasthttp)
and the other provides only rpc over websocket based on two very fast library [gobwas](https://github.com/gobwas/ws) and
[gnet](https://github.com/panjf2000/gnet), however you are not limited to use those, and
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

1. The first need [Go](https://golang.org/) installed (**version 1.17+ is required**), then you can use the below Go command to install
   RonyKIT.

```sh
$ go get -u github.com/clubpay/ronykit/...
```

2. Import it in your code:

```go
package main

import "github.com/clubpay/ronykit"
```

3. (Optional) You might also need to import some standard bundles provided in the main repo.

```go
package main

import (
	"github.com/clubpay/ronykit/std/bundle/fasthttp"
	"github.com/clubpay/ronykit/std/bundle/fastws"
)
```

## Quick start

You can find some sample codes in the [Examples](examples) folder.

## Benchmark

### Simple Echo

We used wrk to benchmark Ronykit standard REST implementation vs Fiber which is fully optimized for REST

RonyKit:

```shell
Running 10s test @ http://127.0.0.1/echo/1200
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     8.22ms    3.15ms  60.00ms   90.54%
    Req/Sec    10.09k     2.04k   18.19k    82.92%
  1208482 requests in 10.05s, 175.18MB read
  Socket errors: connect 0, read 3228, write 0, timeout 0
Requests/sec: 120283.71
Transfer/sec:     17.44MB

```

Fiber:

```shell
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    15.84ms    2.58ms  62.44ms   96.64%
    Req/Sec     5.19k   787.51     5.85k    96.00%
  619345 requests in 10.01s, 73.83MB read
  Socket errors: connect 0, read 3235, write 0, timeout 0
Requests/sec:  61844.21
Transfer/sec:      7.37MB
```

## Define Services and Contracts

In RonyKIT you separate the description of your services and their contracts from the implementation code. For this we have a dedicated
package [desc](desc) which you are encouraged to use it to describe your services.

### Define Service Descriptor

The following code demonstrates the steps you need to take to register your service.

```go
package main

import (
	"encoding/json"
	"syscall"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/desc"
	"github.com/clubpay/ronykit/std/bundle/fasthttp"
)

type echoRequest struct {
	RandomID int64 `json:"randomId" paramName:"randomID"`
}

func (e *echoRequest) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

var serviceDesc = desc.NewService("yourServiceUniqueName").
	Add(
		desc.NewContract().
			// Add the selector which this contract will be selected based of. Every bundle have its own selector 
			// and here we use our std/fasthttp package. 
			AddSelector(fasthttp.Selector{
				Method: fasthttp.MethodGet,
				Path:   "/echo/:randomID",
			}).
			// Define the input message, it defines the message your handler expects to receive.
			SetInput(&echoRequest{}).
			// Set the handler which is going to handle this request.
			SetHandler(echoHandler),
	)

func main() {
	app := ronykit.NewServer(
		ronykit.RegisterService(
			serviceDesc.Generate(),
		),
	)

	// Start the server 
	app.Start()
	
	// Wait for signal to shutdown
	app.Shutdown(syscall.SIGHUP)
}
```