package main

import (
	"context"
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/clusters/rediscluster"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

func main() {
	var (
		instanceID string
		port       int
	)
	flag.StringVar(&instanceID, "instanceID", "", "")
	flag.IntVar(&port, "port", 80, "")
	flag.Parse()

	// Create, start and wait for shutdown signal of the server.
	defer kit.NewServer(
		kit.RegisterCluster(
			instanceID,
			rediscluster.MustNew(
				rediscluster.WithRedisURL("redis://localhost:6380"),
			),
		),
		kit.RegisterGateway(
			fasthttp.MustNew(
				fasthttp.Listen(fmt.Sprintf(":%d", port)),
			),
		),
		desc.Register(genDesc(instanceID)),
	).
		Start(context.TODO()).
		PrintRoutes(os.Stdout).
		Shutdown(context.TODO(), os.Kill, os.Interrupt)

	//nolint:forbidigo
	fmt.Println("Server started.")
}
