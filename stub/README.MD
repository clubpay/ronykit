# STUB

This package provides a set of utilities and helper methods to call a REST or RPC (over websocket)
By using this package many of the boilerplate codes are not necessary anymore.

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
		SetPath("/22fda9e7-1660-406e-b11e-993e070f175e").
		SetQuery("someKey", "someValue").
		Run(ctx)
	defer httpCtx.Release()
}

```
