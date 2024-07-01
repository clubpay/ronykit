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
	"github.com/orlangure/gnomock"
	redisContainer "github.com/orlangure/gnomock/preset/redis"
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
		redisC, err := gnomock.Start(
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
		)

		if err != nil {
			panic(err)
		}
		defer func() {
			err := gnomock.Stop(redisC)
			if err != nil {
				panic(err)
			}
		}()
	}

	// Create, start and wait for shutdown signal of the server.
	defer kit.NewServer(
		kit.WithTrace(
			tracekit.B3("ex03"),
		),
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
