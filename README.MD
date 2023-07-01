# RonyKIT
RonyKIT is a set of tools that are designed to be extendable and flexible to write a complete API/Edge server quickly.
When you develop API handlers using RonyKIT framework, you can support RPC and REST style APIs without re-writing the
endpoint layer multiple times.

Basically when you develop a handler in RonyKIT you just need to think about the input and output of your API, and define
these DTOs by defining appropriate structs.

For example if you want to implement an `Echo` handler you can write your handler like this:
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
        SetMsg(
            &EchoResponse{
                ID: req.ID,
						},
        ).Send()
}
```

Then you can write the ServiceDescriptor to define how your handler is going to be accessed. For example if
your `Echo` request is going to be access using REST API, you can define your service descriptor like this:

```go
package main

import (
    "github.com/clubpay/ronykit/kit"
		"github.com/clubpay/ronykit/kit/desc"
    "github.com/clubpay/ronykit/std/gateways/fasthttp"
)


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
```
In the ServiceDescriptor we defined two REST endpoint which our Handler would be served. RonyKIT's EdgeServer tries its best to fill the
input struct (i.e. EchoRequest) from the parameters: urlParam, queryParam or request's body

The last step is to set up our EdgeServer and bind our desired Gateway bundle and the `MyServiceDesc`
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
		SetMsg(
			&EchoResponse{
				ID: req.ID,
			},
		).Send()
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

# RonyKIT Components

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
# RonyKIT Storage layers
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

---

---
# RonyKIT Standard Packages
This repository is holding all the projects related to RonyKIT. The main package which is the heart of
the RonyKIT framework is [kit](./kit) and the standard modules are in the /std folders.

| Package       | BundleType | Version  | Description                                                                                                                          |
|---------------|------------|----------|--------------------------------------------------------------------------------------------------------------------------------------|
| KIT           | -          | v0.10.26 | the main package of the RonyKIT framework                                                                                            |
| fasthttp      | Gateway    | v0.10.26 | the Gateway bundle implemented using [fasthttp](https://github.com/valyala/fasthttp) framework                                       |
| fastws        | Gateway    | v0.10.26 | the Gateway bundle implemented using [gnet](https://github.com/panjf2000/gnet) and [gobwas](https://github.com/gobwas/ws) frameworks |
| silverhttp    | Gateway    | v0.10.26 | the Gateway bundle implemented using the new kid super-fast http server [silverlining](https://github.com/go-www/silverlining)       |
| rediscluster  | Cluster    | v0.10.26 | the Cluster bundle implemented using [redis](https://github.com/go-redis/redis)                                                      |
| gossipcluster | Cluster    | soon     | Work in progress                                                                                                                     |

