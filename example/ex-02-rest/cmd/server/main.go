package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/debug"
	"syscall"

	"github.com/clubpay/ronykit/example/ex-02-rest/api"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

func main() {
	go func() {
		_ = http.ListenAndServe(":1234", nil)
	}()

	// Create, start and wait for shutdown signal of the server.
	defer kit.NewServer(
		kit.WithPrefork(),
		kit.WithErrorHandler(func(ctx *kit.Context, err error) {
			fmt.Println(err, string(debug.Stack()))
		}),
		kit.WithGateway(
			fasthttp.MustNew(
				fasthttp.Listen(":80"),
				fasthttp.WithServerName("RonyKIT Server"),
				fasthttp.WithCORS(fasthttp.CORSConfig{}),
				fasthttp.WithWebsocketEndpoint("/ws"),
				fasthttp.WithPredicateKey("cmd"),
			),
		),
		kit.WithServiceDesc(
			api.SampleDesc.Desc(),
		),
	).
		Start(context.TODO()).
		PrintRoutes(os.Stdout).
		Shutdown(context.TODO(), syscall.SIGHUP)
}
