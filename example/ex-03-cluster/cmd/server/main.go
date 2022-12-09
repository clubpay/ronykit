package main

import (
	"context"
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/clubpay/ronykit/contrib/tracekit"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/clusters/rediscluster"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

func main() {
	var port int

	flag.IntVar(&port, "port", 80, "")
	flag.Parse()

	// Create, start and wait for shutdown signal of the server.
	defer kit.NewServer(
		kit.WithTrace(
			tracekit.B3("ex03"),
		),
		kit.RegisterCluster(
			rediscluster.MustNew(
				"ex03",
				rediscluster.WithRedisURL("redis://localhost:6380"),
			),
		),
		kit.RegisterGateway(
			fasthttp.MustNew(
				fasthttp.Listen(fmt.Sprintf(":%d", port)),
			),
		),
		kit.RegisterServiceDesc(serviceDesc.Desc()),
	).
		Start(context.TODO()).
		PrintRoutes(os.Stdout).
		Shutdown(context.TODO(), os.Kill, os.Interrupt)
}
