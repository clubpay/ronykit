package testkit

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/clubpay/ronykit/util"
	"github.com/clubpay/ronykit/util/settings"
	_ "github.com/jackc/pgx/v5/stdlib" // required by InitDB
	"github.com/orlangure/gnomock"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func Run(t *testing.T, opts ...fx.Option) {
	t.Helper()
	app := fxtest.New(
		t,
		fx.NopLogger,
		settings.Init,
		fx.Decorate(decorateConfig),
		setupModule(opts...),
	).RequireStart()
	t.Cleanup(app.RequireStop)
}

func decorateConfig(set settings.Settings) settings.Settings {
	_, filename, _, ok := runtime.Caller(0)
	_ = ok

	err := set.SetFromFile("testenv", filepath.Dir(filename))
	if err != nil {
		panic(err)
	}

	return set
}

func setupModule(opts ...fx.Option) fx.Option {
	return fx.Module(
		"testkit",
		fx.Options(opts...),
		fx.Invoke(),
	)
}

type InitDBParams struct {
	// ResultNameTag makes the created sql.DB driver to be tagged with.
	ResultNameTag string // Optional
	User          string
	Pass          string
	DB            string
	Queries       []string
}

func InitDB(containerName string, params InitDBParams) fx.Option {
	return fx.Options(
		fx.Supply(
			util.DynCast[provideDBContainerParams](params),
		),
		fx.Provide(
			fx.Annotate(
				provideDBContainer,
				fx.ResultTags(fmt.Sprintf("name:%q", containerName)),
			),
			fx.Annotate(
				func(c *gnomock.Container) (*sql.DB, error) {
					dsn := fmt.Sprintf(
						"host=%s user=%s password=%s database=%s port=%d sslmode=disable",
						c.Host, params.User, params.Pass, params.DB, c.DefaultPort(),
					)

					db, err := sql.Open("pgx", dsn)
					if err != nil {
						return nil, errors.Wrap(err, "failed to open db")
					}

					err = db.Ping()
					if err != nil {
						return nil, errors.Wrap(err, "failed to ping db")
					}

					return db, nil
				},
				fx.ParamTags(fmt.Sprintf("name:%q", containerName)),
				fx.ResultTags(fmt.Sprintf("name:%q", params.ResultNameTag)),
			),
		),
	)
}

type InitRedisParams struct {
	// ResultNameTag makes the created redis.Client driver to be tagged with.
	ResultNameTag string // Optional
}

func InitRedis(containerName string, params InitRedisParams) fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				provideRedisContainer,
				fx.ResultTags(fmt.Sprintf("name:%q", containerName)),
			),
			fx.Annotate(
				func(c *gnomock.Container) (*redis.Client, error) {
					opt, err := redis.ParseURL("redis://" + net.JoinHostPort(c.Host, util.IntToStr(c.DefaultPort())))
					if err != nil {
						return nil, err
					}

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
				},
				fx.ParamTags(fmt.Sprintf("name:%q", containerName)),
				fx.ResultTags(fmt.Sprintf("name:%q", params.ResultNameTag)),
			),
		),
	)
}
