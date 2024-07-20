# STUB
[![Go Reference](https://pkg.go.dev/badge/github.com/clubpay/ronykit/stub.svg)](https://pkg.go.dev/github.com/clubpay/ronykit/stub)
[![Go Report Card](https://goreportcard.com/badge/github.com/clubpay/ronykit/stub)](https://goreportcard.com/report/github.com/clubpay/ronykit/stub)

This package provides a set of utilities and helper methods to call a REST or RPC (over websocket)
Many of the boilerplate codes are not necessary anymore when you use `stub` and it provides many handy features.

REST example:

```go
package main

import (
	"context"
	"net/http"

	"github.com/clubpay/ronykit/stub"
)

func main() {
	ctx := context.Background()
	s := stub.New("webhook.site", stub.Secure())
	httpCtx := s.REST().
		SetMethod(http.MethodGet).
		SetHeader("Content-Type", "application/json").
		SetPath("/22fda9e7-1660-406e-b11e-993e070f175e").
		SetQuery("someKey", "someValue").
		Run(ctx)
	defer httpCtx.Release()
}

```

## REST
### Response Handler
When you call a REST endpoint, you can use `SetResponseHandler` to handle the response.
There are a few helper methods to write handlers for different response status codes.
Also, you use `DefaultResponseHandler` to set a default handler for all status codes,
which are not handled by other handlers.

```go
package main

import (
	"context"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/stub"
)

type ErrorMessage struct {
	Message string `json:"message"`
}

type SuccessMessage struct {
	Message string `json:"message"`
}

func main() {
	ctx := context.Background()
	s := stub.New("webhook.site", stub.Secure())
	httpCtx := s.REST().
		POST("/someendPoint/slug1").
		SetHeader("Content-Type", "application/json").
		SetQuery("someKey", "someValue").
		SetResponseHandler(
			400,
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {
				res := &ErrorMessage{}
				err := stub.WrapError(kit.UnmarshalMessage(r.GetBody(), res))
				if err != nil {
					return err
				}

				return stub.NewErrorWithMsg(res)
			},
		).
		DefaultResponseHandler(
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {
				res := &SuccessMessage{}

				return stub.WrapError(kit.UnmarshalMessage(r.GetBody(), res))
			},
		).
		Run(ctx)
	defer httpCtx.Release()

  if httpCtx.Error() != nil {
		// handle error
	}
}


```
