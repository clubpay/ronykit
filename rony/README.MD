# Rony

[![Go Reference](https://pkg.go.dev/badge/github.com/clubpay/ronykit/rony.svg)](https://pkg.go.dev/github.com/clubpay/ronykit/rony) [![Go Report Card](https://goreportcard.com/badge/github.com/clubpay/ronykit/rony)](https://goreportcard.com/report/github.com/clubpay/ronykit/rony)

Rony is the high-level API and edge server framework in RonyKIT. It provides a compact, type-safe API with Go generics. The framework uses the `kit` core.

Use the low-level [Kit package](../kit/README.MD) for direct control of gateways or clusters.

- [Installation](#installation)
- [Quick start](#quick-start)
- [Core concepts](#core-concepts)
- [Minimal example](#minimal-example)
- [More examples](#more-examples)

## Installation

Install the project scaffolding tool:

```shell
go install github.com/clubpay/ronykit/ronyup@latest
```

## Quick start

Create a project with `ronyup`:

```shell
ronyup setup -d ./my-project -m github.com/ehsannm/myproject -p MyProjectName
```

## Core concepts

- **State:** Each service can have shared reducer-style state. Implement `Name()` and `Reduce(action) error`. Use `rony.EmptyState()` if the service has no state.
- **Handlers:** Use unary, streaming, or relay functions. Unary handlers receive `*rony.UnaryCtx` and a request DTO. They return a response DTO. Relay handlers receive `*rony.RelayCtx` and return `error` after `ctx.Relay(targetURL, cfg)`.
- **Contracts:** Contracts bind handlers to REST or RPC selectors. Use `WithRelay` for pass-through proxy routes. Do not use `WithUnary` for these routes.

State interface signature:

```go
package rony

type (
	Action          comparable
	State[A Action] interface {
		Name() string
		Reduce(action A) error
	}
)
```

## Minimal example

```go
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/clubpay/ronykit/rony"
	"github.com/clubpay/ronykit/rony/errs"
)

type Counter struct {
	sync.Mutex
	Count int
}

func (c *Counter) Name() string {
	return "Counter"
}

func (c *Counter) Reduce(action string) error {
	switch strings.ToLower(action) {
	case "up":
		c.Count++
	case "down":
		if c.Count <= 0 {
			return fmt.Errorf("count cannot be negative")
		}
		c.Count--
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	return nil
}

type CounterRequest struct {
	Action string `json:"action"`
}

type CounterResponse struct {
	Count int `json:"count"`
}

func count(ctx *rony.UnaryCtx[*Counter, string], req CounterRequest) (*CounterResponse, error) {
	res := &CounterResponse{}
	err := ctx.ReduceState(req.Action, func(s *Counter, err error) error {
		if err != nil {
			return errs.WrapCode(err, errs.InvalidArgument, "BAD_REQUEST")
		}
		res.Count = s.Count
		return nil
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func main() {
	srv := rony.NewServer(
		rony.Listen(":80"),
		rony.WithServerName("CounterServer"),
	)

	rony.Setup(
		srv,
		"CounterService",
		rony.ToInitiateState[*Counter, string](&Counter{Count: 0}),
		rony.WithUnary(
			count,
			rony.GET("/count/{action}"),
			rony.GET("/count"),
		),
	)

	if err := srv.Run(context.Background(), os.Interrupt, os.Kill); err != nil {
		panic(err)
	}
}
```

## More examples

- [Getting Started Guide](../docs/getting-started.md): Build a server and learn core concepts.
- [Cookbook](../docs/cookbook.md): Use patterns for authentication, pagination, and validation.
- [ronyup Guide](../docs/ronyup-guide.md): Use the scaffolding CLI and MCP server.
- [Runnable examples](../example): Run complete example services.
