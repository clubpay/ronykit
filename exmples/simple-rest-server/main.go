package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"syscall"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/desc"
	"github.com/clubpay/ronykit/exmples/simple-rest-server/api"
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
			fmt.Println(ctx, err)
		}),
		ronykit.RegisterBundle(
			fasthttp.MustNew(
				fasthttp.Listen(":80"),
				fasthttp.WithServerName("RonyKIT Server"),
				fasthttp.WithCORS(fasthttp.CORSConfig{}),
				fasthttp.WithWebsocketEndpoint("/ws"),
				fasthttp.WithPredicateKey("cmd"),
				fasthttp.WithReflectTag("json"),
			),
		),
		desc.Register(api.NewSample()),
	).
		Start(nil).
		PrintRoutes(os.Stdout).
		Shutdown(nil, syscall.SIGHUP)

	//nolint:forbidigo
	fmt.Println("Server started.")
}
