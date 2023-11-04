# Rony

Rony is a framework for developing high-performance API servers.
It is designed to be simple and flexible.
It uses the power of Go generics to provide an easy and robust code and helps you to detect
a good few of the common mistakes at compile time.
If you need to have more control over your code, such as selecting your own Gateway and Cluster,
you can use [Kit Package](../kit/README.MD) however for most use cases we recommend to use `rony` package instead.

- [Installation](#installation)
- [Quick start](#quick-start)
  - [State](#state)
  - [Counter Program with State](#counter-program-with-state)


## Installation

To install Rony package, you need to install Go and set your Go workspace first.

First need to install[Go](https://golang.org/) (**version 1.19+ is required**), then you can use the below Go command to install
RonyKIT.

```sh
$ go get -u github.com/clubpay/ronykit/rony/...
```

## Quick start

### State
When you develop API servers, you usually have a common state which could be shared between a few of your endpoints (i.e., Contracts).
For example, you may have a database connection or a cache connection which you want to share between your endpoints.
Moreover, you may want to have a shared state like a counter of the requests you received or in a simple chat application
to keep the list of the connected users. Rony lets you define your own state and provides it to your handlers, so you can
access it from your handlers without relying on global variables or defining your handler functions as methods of a common
structs. The later two approaches can be problematic as your project grows.

The following code shows the type parameter of the State that your need to implement for your server.
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

As you can see, the State is a generic type which has a type parameter named `Action` which is a comparable type. This is
defined to let you define your state in a `Reducer pattern`. Also we recommend that your state also implements sync.Locker interface
to be thread-safe.

### Counter Program with State
Lets first implement our State. We want to have a simple counter which counts the number of requests we received. Our `EchoCounter` state
has action type of `string` and it supports two actions: `up` and `down`. The `up` action increases the counter and the `down` action
decreases the counter. The following code shows the implementation of the `EchoCounter` state.

```go
package main

import (
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

func (e *EchoCounter) Reduce(action string) {
	switch action {
	case "up":
		e.Count++
	case "down":
		e.Count--
	}
}

```

Now We want to implement our handlers to handle UP and DOWN functionality. We need to first define DTOs (Data Transfer Objects) of our
handlers. However instead of defining two separate DTOs, we going to define one DTO and use the `Action` field of the DTO to determine
the action we want to perform. Although, this is just in our example, in your case you may come up with a better design.

```go
package main

import (
	"github.com/clubpay/ronykit/rony"
)

type CounterRequestDTO struct {
	Action string `json:"action"`
	Count int `json:"count"`
}

type CounterResponseDTO struct {
	Count int `json:"count"`
}
```

Now we need to define our handler. Let's first define our handler struct.

```go
package main

func count(ctx *rony.UnaryCtx[*EchoCounter, string], req *CounterRequestDTO) (*CounterResponseDTO, error) {
	res := &CounterResponseDTO{}
	err := ctx.ReduceState(
		req.Action,
		func(s *EchoCounter) error {
			if s.Count < 0 {
				return rony.NewError(fmt.Errorf("count is negative: %d", s.Count)).SetCode(http.StatusBadRequest)
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

As you can see, we have a function named `count` which has two parameters. The first parameter is a `UnaryCtx` which is a generic type
and provides the state to the handler. It also has many more helper methods which we will discuss later. The second parameter is the
request DTO. The return value of the handler is the response DTO and an error. The handler function is a normal function and it is not
a method of a struct.
Inside the handler code, you see we have `ReduceState` method which is a helper method that let us to mutate the state in an atomic
fashion. The code in the callback function of the `ReduceState` method is executed in a thread-safe manner and it is guaranteed that
no other goroutine is mutating the state while the callback function is running.

Now let's wrap up the code and define our server.

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

	// Set up the server with the initial state, which is a pointer to EchoCounter
	// We can have as many states as we want. But each handler can only work with
	// one state. In other words, we cannot register one handler with two different
	// setup contexts.
	setupCtx := rony.Setup(
		srv,
		rony.ToInitiateState[*EchoCounter, string](
			&EchoCounter{
				Count: 0,
			},
		),
	)

	// Register the count handler for both GET /count and GET /count/{action}
	// This way all the following requests are valid:
	// 1. GET /count/up&count=1
	// 2. GET /count/down&count=2
	// 3. GET /count?action=up&count=1
	// 4. GET /count?action=down&count=2
	rony.RegisterUnary(
		setupCtx, count,
		rony.GET("/count/{action}"),
		rony.GET("/count"),
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

We first create a new server and then we set up the server with the initial state. We can have as many states as we want.
Then we register our handler with the server. We can register as many handlers as we want. Finally we run the server in blocking mode.
You can also check examples to see [more examples](./examples).

