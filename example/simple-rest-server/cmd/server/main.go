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

	"github.com/clubpay/ronykit/example/simple-rest-server/api"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/pkg/profile"
)

func main() {
	defer profile.Start(profile.MemProfile, profile.ProfilePath(".")).Stop()
	runtime.GOMAXPROCS(4)

	go func() {
		_ = http.ListenAndServe(":1234", nil)
	}()

	// Create, start and wait for shutdown signal of the server.
	defer kit.NewServer(
		kit.WithErrorHandler(func(ctx *kit.Context, err error) {
			fmt.Println(err, string(debug.Stack()))
		}),
		kit.RegisterGateway(
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
