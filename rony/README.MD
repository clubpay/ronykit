# Rony
[![Go Reference](https://pkg.go.dev/badge/github.com/clubpay/ronykit/rony.svg)](https://pkg.go.dev/github.com/clubpay/ronykit/rony)
[![Go Report Card](https://goreportcard.com/badge/github.com/clubpay/ronykit/rony)](https://goreportcard.com/report/github.com/clubpay/ronykit/rony)

Rony is a framework for developing high-performance API servers. It is designed to be simple and flexible,
leveraging the power of Go generics to provide an easy-to-use and robust codebase that helps detect common
mistakes at compile time. If you need more control over your code, such as selecting your own Gateway and
Cluster, you can use the [Kit Package](../kit/README.MD). However, for most use cases, we recommend using
the `rony` package.

- [Installation](#installation)
- [Quick Start](#quick-start)
	- [State](#state)
	- [Counter Program with State](#counter-program-with-state)


## Quick Start
To get started with RonyKIT, you can use the following command to install the `ronyup` cli tool:

```shell
go install github.com/clubpay/ronykit/ronyup@latest
```

After installing the `ronyup` tool, you can create a new project using the `ronyup setup` command, below is an example:

```shell
ronyup setup -d ./my-project -m github.com/ehsannm/myproject -p MyProjectName
```

## Getting Started
### State

When developing API servers, you often have a common state that can be shared between several of your
endpoints (i.e., Contracts). For example, you might have a database or cache connection that you want to
share between your endpoints. Additionally, you might want a shared state like a counter for the requests
received or, in a simple chat application, to keep a list of connected users. Rony allows you to define
your own state and provide it to your handlers, enabling access without relying on global variables or
defining handler functions as methods of common structs. These approaches can be problematic as your project grows.

The following code shows the type parameter of the State that you need to implement for your server.

```go
package rony

// State related types
type (
  Action          comparable
  State[A Action] interface {
    Name() string
    Reduce(action A)
  }
)
```

The State is a generic type with a type parameter named `Action`, which is a comparable type. This design allows you to
define your state in a `Reducer pattern`. We also recommend that your state implements the sync.Locker interface to
be thread-safe.

### Counter Program with State

Let's first implement our State. We want a simple counter that counts the number of requests received. Our `EchoCounter` state
has an action type of `string` and supports two actions: `up` and `down`. The `up` action increases the counter,
and the `down` action decreases it. The following code shows the implementation of the `EchoCounter` state.

```go
package main

import (
  "fmt"
  "strings"
  "sync"

  "github.com/clubpay/ronykit/rony"
)

type EchoCounter struct {
  sync.Mutex
  Count int
}

func (e *EchoCounter) Name() string {
  return "EchoCounter"
}

func (e *EchoCounter) Reduce(action string) error {
  switch strings.ToLower(action) {
  case "up":
    e.Count++
  case "down":
    if e.Count <= 0 {
      return fmt.Errorf("count cannot be negative")
    }
    e.Count--
  default:
    return fmt.Errorf("unknown action: %s", action)
  }
  return nil
}
```

Next, we need to implement our handlers to handle UP and DOWN functionality. We'll define DTOs (Data Transfer Objects) for
our handlers. Instead of defining two separate DTOs, we'll define one DTO and use the `Action` field to determine the
action to perform.

```go
package main

type CounterRequestDTO struct {
  Action string `json:"action"`
  Count int `json:"count"`
}

type CounterResponseDTO struct {
  Count int `json:"count"`
}
```

Now we define our handler.

```go
package main

import (
  "github.com/clubpay/ronykit/rony"
)

func count(ctx *rony.UnaryCtx[*EchoCounter, string], req *CounterRequestDTO) (*CounterResponseDTO, error) {
  res := &CounterResponseDTO{}
  err := ctx.ReduceState(
    req.Action,
    func(s *EchoCounter, err error) error {
      if err != nil {
        return rony.NewError(err).SetCode(http.StatusBadRequest)
      }
      res.Count = s.Count
      return nil
    },
  )
  if err != nil {
    return nil, err
  }
  return res, nil
}
```

Our handler function, `count`, has two parameters. The first is a `UnaryCtx`, a generic type that provides the state
to the handler, along with many helper methods. The second parameter is the request DTO. The handler returns the
response DTO and an error. The `ReduceState` method in the handler allows us to mutate the state in an atomic fashion.
The code in the `ReduceState` callback function executes in a thread-safe manner, ensuring no other goroutine mutates
the state simultaneously.

Finally, let's wrap up the code and define our server.

```go
package main

import (
  "context"
  "os"

  "github.com/clubpay/ronykit/rony"
)

func main() {
  srv := rony.NewServer(
    rony.Listen(":80"),
    rony.WithServerName("CounterServer"),
  )

  // Set up the server with the initial state, a pointer to EchoCounter
  // We can have multiple states, but each handler works with only one state.
  // In other words, we cannot register one handler with two different states.
  rony.Setup(
    srv,
    "CounterService",
    rony.ToInitiateState[*EchoCounter, string](
      &EchoCounter{
        Count: 0,
      },
    ),
    // Register the count handler for both GET /count and GET /count/{action}
    // This way, the following requests are valid:
    // 1. GET /count/up&count=1
    // 2. GET /count/down&count=2
    // 3. GET /count?action=up&count=1
    // 4. GET /count?action=down&count=2
    rony.WithUnary(
      count,
      rony.GET("/count/{action}"),
      rony.GET("/count"),
    ),
  )

  // Run the server in blocking mode
  err := srv.Run(
    context.Background(),
    os.Kill, os.Interrupt,
  )
  if err != nil {
    panic(err)
  }
}
```

We first create a new server and then set up the server with the initial state. Multiple states can be set up, but each
handler works with only one state. Then we register our handler with the server, allowing multiple handlers to be registered.
Finally, we run the server in blocking mode. For more examples, check the [examples](./examples) directory.
