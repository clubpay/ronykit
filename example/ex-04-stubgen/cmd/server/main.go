package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/clubpay/ronykit/example/ex-04-stubgen/api"
	"github.com/clubpay/ronykit/example/ex-04-stubgen/stub/sampleservice"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

func main() {
	go func() {
		_ = http.ListenAndServe(":1234", nil)
	}()

	go func() {
		time.Sleep(time.Second * 3)
		s := sampleservice.NewsampleServiceStub("127.0.0.1:80")
		res, err := s.ComplexDummy2(context.Background(), &sampleservice.VeryComplexRequest{
			Key1: "someting",
		})
		if err != nil {
			fmt.Println("ERR", err)
		} else {
			fmt.Println("OK", res)
		}
	}()

	// Create, start and wait for shutdown signal of the server.
	defer kit.NewServer(
		//kit.WithPrefork(),
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
