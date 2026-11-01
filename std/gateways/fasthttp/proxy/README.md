# fasthttp-reverse-proxy

![](https://img.shields.io/badge/LICENSE-MIT-blue.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/yeqown/fasthttp-reverse-proxy/v2)](https://goreportcard.com/report/github.com/yeqown/fasthttp-reverse-proxy/v2) [![GoReportCard](https://godoc.org/github.com/yeqown/fasthttp-reverse-proxy/v2?status.svg)](https://godoc.org/github.com/yeqown/fasthttp-reverse-proxy/v2)

This package provides a reverse HTTP proxy handler based on fasthttp.

## Features

- [X] HTTP reverse proxy based on [fasthttp](https://github.com/valyala/fasthttp)
  - [X] Faster than the Go standard `httputil.ReverseProxy` library
  - [X] Implemented with `fasthttp.HostClient`
  - [X] Round-robin load balancing
  - [X] `HostClient` object pool over the fasthttp connection pool
- [X] WebSocket reverse proxy

## Get started

### [HTTP with balancer option](https://github.com/yeqown/fasthttp-reverse-proxy/blob/master/examples/fasthttp-reverse-proxy-with-bla/proxy.go)

```go
var (
	proxyServer = proxy.NewReverseProxy("localhost:8080")

	// use with balancer
	// weights = map[string]proxy.Weight{
	// 	"localhost:8080": 20,
	// 	"localhost:8081": 30,
	// 	"localhost:8082": 50,
	// }
	// proxyServer = proxy.NewReverseProxy("", proxy.WithBalancer(weights))

)

// ProxyHandler ... fasthttp.RequestHandler func
func ProxyHandler(ctx *fasthttp.RequestCtx) {
	// all proxy to localhost
	proxyServer.ServeHTTP(ctx)
}

func main() {
	if err := fasthttp.ListenAndServe(":8081", ProxyHandler); err != nil {
		log.Fatal(err)
	}
}
```

### [WebSocket](https://github.com/yeqown/fasthttp-reverse-proxy/blob/master/examples/ws-fasthttp-reverse-proxy/README.md)

```go
var (
	proxyServer *proxy.WSReverseProxy
	once        sync.Once
)

// ProxyHandler ... fasthttp.RequestHandler func
func ProxyHandler(ctx *fasthttp.RequestCtx) {
	once.Do(func() {
		var err error
		proxyServer, err = proxy.NewWSReverseProxyWith(
			proxy.WithURL_OptionWS("ws://localhost:8080/echo"),
		)
		if err != nil {
			panic(err)
		}
	})

	switch string(ctx.Path()) {
	case "/echo":
		proxyServer.ServeHTTP(ctx)
	case "/":
		fasthttp.ServeFileUncompressed(ctx, "./index.html")
	default:
		ctx.Error("Unsupported path", fasthttp.StatusNotFound)
	}
}

func main() {
	log.Println("serving on: 8081")
	if err := fasthttp.ListenAndServe(":8081", ProxyHandler); err != nil {
		log.Fatal(err)
	}
}

```

## Usage examples

- [HTTP reverse proxy](https://github.com/yeqown/fasthttp-reverse-proxy/blob/master/examples/fasthttp-reverse-proxy/proxy.go)
- [HTTP reverse proxy with object pool](https://github.com/yeqown/fasthttp-reverse-proxy/blob/master/examples/fasthttp-reverse-proxy-with-pool/pool.go)
- [WebSocket reverse proxy](https://github.com/yeqown/fasthttp-reverse-proxy/tree/master/examples/ws-fasthttp-reverse-proxy)

## Benchmarks

- [HTTP benchmark](https://github.com/yeqown/fasthttp-reverse-proxy/blob/master/docs/http-benchmark.md)
- [WebSocket benchmark](https://github.com/yeqown/fasthttp-reverse-proxy/blob/master/docs/ws-benchmark.md)

## References

- [fasthttp](https://github.com/valyala/fasthttp)
- [standard httputil.ReverseProxy](https://golang.org/pkg/net/http/httputil/#ReverseProxy)
- [fasthttp/websocket](https://github.com/fasthttp/websocket)
- [koding/websocketproxy](https://github.com/koding/websocketproxy)

## Handler relay and gateway reverse proxy in RonyKIT

RonyKIT provides two complementary mechanisms:

| Mechanism | Location | Target | Registration |
|-----------|----------|--------|--------------|
| **Gateway reverse proxy** | Gateway router before contracts | Fixed upstream addresses | `rony.WithReverseProxy(path, proxy.WithAddress(...))` or `fasthttp.WithReverseProxy` |
| **Handler relay** | Contract handler | Dynamic `targetURL` string for each request | `rony.WithRelay` with `RelayCtx.Relay()` or `kit.Relay` on `*kit.Context` |

Use **gateway reverse proxy** for static assets or a known backend path.

Use **handler relay** when the handler must process the request before forwarding. Examples include session validation, dynamic address resolution, path rewrites, and sensitive query parameter removal.

Implementation details:

- `std/gateways/fasthttp` implements handler relay on `*httpConn` with `kit.RelayConn`.
- `proxy.HopHeaders()` supplies the shared hop-by-hop header list.
- Dynamic relay entry points are `proxy.RelayHTTP` and `proxy.RelayWebSocket`. Application code must use `kit.Relay` or `RelayCtx.Relay` instead.
- A successful relay returns `kit.ErrRelayCompleted` and stops the handler chain. It does not write a JSON envelope to the client.

See also ronyup MCP resource `architecture/handler-relay`, `kit/CHANGELOG.md`, and the handler relay section in `docs/cookbook.md`.

## Thanks

<a href="https://www.jetbrains.com/?from=fasthttp-reverse-proxy" _blank="#">
    <img src="https://www.jetbrains.com/company/brand/img/jetbrains_logo.png" width="100" alt="JetBrains"/>
</a>
