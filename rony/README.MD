# Rony

[![Go Reference](https://pkg.go.dev/badge/github.com/clubpay/ronykit/rony.svg)](https://pkg.go.dev/github.com/clubpay/ronykit/rony)
[![Go Report Card](https://goreportcard.com/badge/github.com/clubpay/ronykit/rony)](https://goreportcard.com/report/github.com/clubpay/ronykit/rony)

Rony is the batteries-included API and edge server framework in RonyKIT. It provides a compact, type-safe API
powered by Go generics and built on top of the `kit` core. If you need full control over gateways or clusters,
use the lower-level [Kit package](../kit/README.MD).

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Minimal Example](#minimal-example)
- [More Examples](#more-examples)

## Installation

Install the project scaffold tool:

```shell
go install github.com/clubpay/ronykit/ronyup@latest
```

## Quick Start

Create a new project with `ronyup`:

```shell
ronyup setup -d ./my-project -m github.com/ehsannm/myproject -p MyProjectName
```

## Core Concepts

- **State**: a shared, reducer-style state per service. Implement `Name()` and `Reduce(action) error`.
  If you do not need state, use `rony.EmptyState()`.
- **Handlers**: unary or streaming functions. Unary handlers receive `*rony.UnaryCtx` and a request DTO,
  then return a response DTO.
- **Contracts**: route definitions that bind handlers to REST or RPC selectors.

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

## Minimal Example

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

## More Examples

- [Getting Started Guide](../GETTING_STARTED.MD)
- [Runnable examples](../example)
