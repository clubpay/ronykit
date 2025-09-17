package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/clubpay/ronykit/contrib/swagger"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/clubpay/ronykit/std/gateways/fastws"
)

func main() {
	runtime.GOMAXPROCS(4)

	go func() {
		_ = http.ListenAndServe(":1234", nil)
	}()

	docUI, _ := swagger.New("Example", "v0.1.2", "").
		ReDocUI(desc.ToDesc(sampleService)...)

	// Create, start and wait for shutdown signal of the server.
	defer kit.NewServer(
		kit.WithErrorHandler(
			func(ctx *kit.Context, err error) {
				fmt.Println("got error: ", err)
			},
		),
		kit.WithGateway(
			fastws.MustNew(
				fastws.WithLogger(common.NewStdLogger()),
				fastws.Listen("tcp4://0.0.0.0:80"),
				fastws.WithPredicateKey("cmd"),
			),
			// We can use fastws Gateway if we have millions of connections which are not very active.
			// fastws ONLY supports websocket, and if you need to handle websocket and http, you should
			// use fasthttp instead.
			//
			fasthttp.MustNew(
				fasthttp.WithLogger(common.NewStdLogger()),
				fasthttp.WithWebsocketEndpoint("/"),
				fasthttp.WithPredicateKey("cmd"),
				fasthttp.Listen(":81"),
				fasthttp.WithServeFS("/api-doc", "", docUI),
			),
		),
		kit.WithServiceBuilder(sampleService),
	).
		Start(context.TODO()).
		PrintRoutesCompact(os.Stdout).
		Shutdown(context.TODO(), os.Kill, os.Interrupt)
}
