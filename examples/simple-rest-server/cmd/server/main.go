package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/desc"
	"github.com/clubpay/ronykit/examples/simple-rest-server/api"
	"github.com/clubpay/ronykit/std/gateway/fasthttp"
)

func main() {
	runtime.GOMAXPROCS(4)

	go func() {
		_ = http.ListenAndServe(":1234", nil)
	}()

	// Create, start and wait for shutdown signal of the server.
	defer ronykit.NewServer(
		ronykit.WithErrorHandler(func(ctx *ronykit.Context, err error) {
			fmt.Println(err, string(debug.Stack()))
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
		desc.Register(api.SampleDesc),
	).
		Start(context.TODO()).
		PrintRoutes(os.Stdout).
		Shutdown(context.TODO(), syscall.SIGHUP)

	//nolint:forbidigo
	fmt.Println("Server started.")
}
