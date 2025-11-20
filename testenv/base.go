package testenv

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/std/clusters/p2pcluster"
	"github.com/clubpay/ronykit/std/clusters/rediscluster"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/clubpay/ronykit/std/gateways/fastws"
	"github.com/orlangure/gnomock"
	redisContainer "github.com/orlangure/gnomock/preset/redis"
	"github.com/redis/go-redis/v9"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

var (
	redisDSN         string
	temporalHostPort string
	TemporalUI       string
)

func getRedis() (*redis.Client, error) {
	opt, err := redis.ParseURL(redisDSN)
	if err != nil {
		return nil, err
	}

	opt.ClientName = "ronykit-test"
	if opt.TLSConfig != nil {
		opt.TLSConfig.MinVersion = tls.VersionTLS12
	}

	cli := redis.NewClient(opt)

	ctx, cf := context.WithTimeout(context.Background(), time.Second*5)
	defer cf()

	err = cli.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func provideRedis(lc fx.Lifecycle) (*redis.Client, error) {
	redisC, err := gnomock.Start(
		redisContainer.Preset(redisContainer.WithVersion("7-alpine")),
		gnomock.WithUseLocalImagesFirst(),
	)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return gnomock.Stop(redisC)
		},
	})

	redisDSN = fmt.Sprintf("redis://localhost:%d", redisC.DefaultPort())

	opt, err := redis.ParseURL(redisDSN)
	if err != nil {
		return nil, err
	}

	opt.ClientName = "ronykit-test"
	if opt.TLSConfig != nil {
		opt.TLSConfig.MinVersion = tls.VersionTLS12
	}

	cli := redis.NewClient(opt)

	ctx, cf := context.WithTimeout(context.Background(), time.Second*5)
	defer cf()

	err = cli.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func invokeTemporal(lc fx.Lifecycle) error {
	options := []gnomock.Option{
		gnomock.WithUseLocalImagesFirst(),
		gnomock.WithEntrypoint(
			"temporal", "server", "start-dev",
			"-n", "kitTest",
			"--ip", "0.0.0.0",
		),
		// gnomock.WithLogWriter(os.Stdout),
		gnomock.WithHealthCheck(
			func(ctx context.Context, container *gnomock.Container) error {
				timeout := time.Second * 3

				conn, err := net.DialTimeout("tcp", container.DefaultAddress(), timeout)
				if err != nil {
					return err
				}

				_ = conn.Close()

				return nil
			}),
	}

	c, err := gnomock.StartCustom(
		"temporalio/server",
		gnomock.NamedPorts{
			gnomock.DefaultPort: gnomock.TCP(7233),
			"ui":                gnomock.TCP(8233),
		},
		options...,
	)
	if err != nil {
		return err
	}

	temporalHostPort = c.DefaultAddress()
	TemporalUI = fmt.Sprintf("http://%s", c.Address("ui"))

	lc.Append(
		fx.Hook{
			OnStop: func(ctx context.Context) error {
				err = gnomock.Stop(c)
				if err != nil {
					return err
				}

				return nil
			},
		})

	return nil
}

func invokeRedisMonitor(lc fx.Lifecycle, _ *redis.Client) {
	out := make(chan string, 10)
	cli, _ := getRedis()
	cmd := cli.Monitor(context.Background(), out)
	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				cmd.Start()

				return nil
			},
			OnStop: func(ctx context.Context) error {
				cmd.Stop()
				close(out)

				return nil
			},
		},
	)

	go func() {
		for x := range out {
			fmt.Println("REDIS:", x)
		}

		fmt.Println("REDIS CLOSED")
	}()
}

func invokeEdgeServerFastHttp(_ string, port int, desc ...kit.ServiceBuilder) fx.Option {
	return fx.Invoke(
		func(lc fx.Lifecycle, _ *redis.Client) {
			edge := kit.NewServer(
				kit.WithLogger(common.NewStdLogger()),
				kit.WithErrorHandler(
					func(ctx *kit.Context, err error) {
						fmt.Println("EdgeError: ", err)
					},
				),
				kit.WithGateway(
					fasthttp.MustNew(
						fasthttp.Listen(fmt.Sprintf(":%d", port)),
						fasthttp.WithWebsocketEndpoint("/agent/ws"),
						fasthttp.WithPredicateKey("cmd"),
						fasthttp.WithLogger(common.NewStdLogger()),
					),
				),
				kit.WithServiceBuilder(desc...),
			)

			lc.Append(
				fx.Hook{
					OnStart: func(ctx context.Context) error {
						edge.PrintRoutesCompact(os.Stdout)
						edge.Start(ctx)

						return nil
					},
					OnStop: func(ctx context.Context) error {
						edge.Shutdown(ctx)

						return nil
					},
				},
			)
		},
	)
}

func invokeEdgeServerWithFastWS(port int, desc ...kit.ServiceBuilder) fx.Option {
	return fx.Invoke(
		func(lc fx.Lifecycle) {
			edge := kit.NewServer(
				kit.ReusePort(true),
				kit.WithLogger(common.NewStdLogger()),
				kit.WithErrorHandler(
					func(ctx *kit.Context, err error) {
						fmt.Println("EdgeError: ", err)
					},
				),
				kit.WithGateway(
					fastws.MustNew(
						fastws.WithPredicateKey("cmd"),
						fastws.Listen(fmt.Sprintf("tcp4://0.0.0.0:%d", port)),
						fastws.WithLogger(common.NewStdLogger()),
					),
				),
				kit.WithServiceBuilder(desc...),
			)

			lc.Append(
				fx.Hook{
					OnStart: func(ctx context.Context) error {
						edge.Start(ctx)

						return nil
					},
					OnStop: func(ctx context.Context) error {
						edge.Shutdown(ctx)

						return nil
					},
				},
			)
		},
	)
}

func invokeEdgeServerWithRedis(_ string, port int, desc ...kit.ServiceBuilder) fx.Option {
	return fx.Invoke(
		func(lc fx.Lifecycle, _ *redis.Client) {
			edge := kit.NewServer(
				kit.WithCluster(
					rediscluster.MustNew(
						"testCluster",
						rediscluster.WithRedisClient(utils.Must(getRedis())),
						rediscluster.WithGCPeriod(time.Second*3),
					),
				),
				kit.WithLogger(common.NewStdLogger()),
				kit.WithErrorHandler(
					func(ctx *kit.Context, err error) {
						fmt.Println("EdgeError: ", err)
					},
				),
				kit.WithGateway(
					fasthttp.MustNew(
						fasthttp.WithDisableHeaderNamesNormalizing(),
						fasthttp.Listen(fmt.Sprintf(":%d", port)),
					),
				),
				kit.WithServiceBuilder(desc...),
			)

			lc.Append(
				fx.Hook{
					OnStart: func(ctx context.Context) error {
						edge.Start(ctx)

						return nil
					},
					OnStop: func(ctx context.Context) error {
						edge.Shutdown(ctx)

						return nil
					},
				},
			)
		},
	)
}

func invokeEdgeServerWithP2P(_ string, port int, desc ...kit.ServiceBuilder) fx.Option {
	return fx.Invoke(
		func(lc fx.Lifecycle, _ *redis.Client) {
			edge := kit.NewServer(
				kit.WithCluster(
					p2pcluster.New(
						"testCluster",
						p2pcluster.WithLogger(common.NewStdLogger()),
						p2pcluster.WithBroadcastInterval(time.Second),
					),
				),
				kit.WithLogger(common.NewStdLogger()),
				kit.WithErrorHandler(
					func(ctx *kit.Context, err error) {
						fmt.Println("EdgeError: ", err)
					},
				),
				kit.WithGateway(
					fasthttp.MustNew(
						fasthttp.WithDisableHeaderNamesNormalizing(),
						fasthttp.Listen(fmt.Sprintf(":%d", port)),
					),
				),
				kit.WithServiceBuilder(desc...),
			)

			lc.Append(
				fx.Hook{
					OnStart: func(ctx context.Context) error {
						edge.Start(ctx)

						return nil
					},
					OnStop: func(ctx context.Context) error {
						edge.Shutdown(ctx)

						return nil
					},
				},
			)
		},
	)
}

func Prepare(t *testing.T, c C, option ...fx.Option) {
	_, _ = c.Println('\n')
	opts := []fx.Option{
		fx.StartTimeout(time.Minute * 5),
		fx.StopTimeout(time.Minute * 5),
		fx.NopLogger,
		fx.Provide(provideRedis),
		fx.Invoke(invokeTemporal),
	}

	opts = append(opts, option...)

	c.Reset(fxtest.New(t, opts...).RequireStart().RequireStop)
}

type kitLogger struct{}

func (k kitLogger) Debugf(format string, args ...any) {
	fmt.Println("kitLogger: ", fmt.Sprintf(format, args...))
}

func (k kitLogger) Errorf(format string, args ...any) {
	fmt.Println("kitLogger: ", fmt.Sprintf(format, args...))
}
