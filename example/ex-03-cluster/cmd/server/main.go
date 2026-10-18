package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/clusters/rediscluster"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/clubpay/ronykit/x/rkit"
	"github.com/clubpay/ronykit/x/telemetry/tracekit"

	"github.com/orlangure/gnomock"
	redisContainer "github.com/orlangure/gnomock/preset/redis"

	_ "net/http/pprof"
)

func main() {
	var (
		port  int
		redis int
	)

	flag.IntVar(&port, "port", 80, "")
	flag.IntVar(&redis, "redis", 0, "")
	flag.Parse()

	if redis > 0 {
		// Create a Redis container.
		redisC := rkit.Must(gnomock.Start(
			redisContainer.Preset(
				redisContainer.WithVersion("7-alpine"),
			),
			gnomock.WithUseLocalImagesFirst(),
			gnomock.WithOptions(&gnomock.Options{
				CustomNamedPorts: map[string]gnomock.Port{
					"default": {
						Protocol: "tcp",
						Port:     6379,
						HostPort: 6379,
					},
				},
			}),
		))

		defer func() {
			rkit.Assert(gnomock.Stop(redisC))
		}()
	}

	// Create, start and wait for shutdown signal of the server.
	defer kit.NewServer(
		kit.WithTrace(
			tracekit.B3("ex03"),
		),
		kit.WithErrorHandler(func(ctx *kit.Context, err error) {
			fmt.Println("error:", err)
		}),
		kit.WithCluster(
			rediscluster.MustNew(
				"ex03",
				rediscluster.WithRedisURL("redis://localhost:6379"),
			),
		),
		kit.WithGateway(
			fasthttp.MustNew(
				fasthttp.Listen(fmt.Sprintf(":%d", port)),
			),
		),
		kit.WithServiceBuilder(serviceDesc.Desc()),
	).
		Start(context.TODO()).
		PrintRoutes(os.Stdout).
		Shutdown(context.TODO(), os.Kill, os.Interrupt)
}
