package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/exmples/simple-rest-server/api"
	"github.com/clubpay/ronykit/std/gateway/fasthttp"
)

func main() {
	runtime.GOMAXPROCS(4)

	// Create, start and wait for shutdown signal of the server.
	defer ronykit.NewServer(
		ronykit.WithErrorHandler(func(ctx *ronykit.Context, err error) {
			fmt.Println(ctx, err)
		}),
		ronykit.RegisterBundle(
			fasthttp.MustNew(
				fasthttp.Listen(":80"),
				fasthttp.WithServerName("RonyKIT Server"),
				fasthttp.WithCORS(fasthttp.CORSConfig{}),
				fasthttp.WithWebsocketEndpoint("/ws"),
				fasthttp.WithPredicateKey("cmd"),
			),
		),
		ronykit.RegisterServiceDesc(
			api.NewSample().Desc(),
		),
	).
		Start(nil).
		PrintRoutes(os.Stdout).
		Shutdown(nil, syscall.SIGHUP)

	//nolint:forbidigo
	fmt.Println("Server started.")
}
