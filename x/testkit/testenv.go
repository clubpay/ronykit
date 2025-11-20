package testkit

import (
	"context"
	"embed"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/clubpay/ronykit/util"
	"github.com/orlangure/gnomock"
	pgContainer "github.com/orlangure/gnomock/preset/postgres"
	redisContainer "github.com/orlangure/gnomock/preset/redis"
	"github.com/pkg/errors"
	"go.uber.org/fx"
)

type provideDBContainerParams struct {
	User    string
	Pass    string
	DB      string
	Queries []string
}

func provideDBContainer(lc fx.Lifecycle, in provideDBContainerParams) (*gnomock.Container, error) {
	dbC, err := gnomock.Start(
		pgContainer.Preset(
			pgContainer.WithUser(in.User, in.Pass),
			pgContainer.WithDatabase(in.DB),
			pgContainer.WithQueries(in.Queries...),
		),
		gnomock.WithUseLocalImagesFirst(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start db container")
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return gnomock.Stop(dbC)
		},
	})

	return dbC, nil
}

func provideRedisContainer(lc fx.Lifecycle) (*gnomock.Container, error) {
	redisC, err := gnomock.Start(
		redisContainer.Preset(),
		gnomock.WithUseLocalImagesFirst(),
	)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return gnomock.Stop(redisC)
		},
	})

	return redisC, nil
}

type WireMockFS struct {
	FS   embed.FS
	Root string
}

//nolint:unused
type wireMockConfig struct {
	fx.In

	MockFS *WireMockFS `name:"wiremockFS" optional:"true"`
}

//nolint:unused
func provideWireMock(lc fx.Lifecycle, cfg wireMockConfig) (*gnomock.Container, error) {
	options := []gnomock.Option{
		gnomock.WithUseLocalImagesFirst(),
		// gnomock.WithHealthCheck(
		//    func(ctx context.Context, container *gnomock.Container) error {
		//        return stub.New(container.DefaultAddress()).
		//            REST().
		//            GET("/_admin/health").
		//            Run(context.Background()).
		//            Error()
		//    }),
	}

	tempFolder := filepath.Join(os.TempDir(), "testkit", util.RandomID(10))
	if cfg.MockFS != nil {
		err := copyWireMockFS(cfg.MockFS, tempFolder)
		if err != nil {
			return nil, err
		}

		options = append(options, gnomock.WithHostMounts(tempFolder, "/home/wiremock"))
	}

	c, err := gnomock.StartCustom(
		"wiremock/wiremock:latest",
		gnomock.NamedPorts{
			gnomock.DefaultPort: gnomock.TCP(8080),
			"securePort":        gnomock.TCP(8443),
		},
		options...,
	)
	if err != nil {
		return nil, err
	}

	lc.Append(
		fx.Hook{
			OnStop: func(_ context.Context) error {
				err = gnomock.Stop(c)
				if err != nil {
					return err
				}

				if cfg.MockFS != nil {
					return os.RemoveAll(tempFolder)
				}

				return nil
			},
		},
	)

	return c, nil
}

//nolint:unused
func copyWireMockFS(mockFS *WireMockFS, tempFolder string) error {
	orgFolders := []string{"mappings", "files"}
	dstFolders := []string{"mappings", "__files"}

	for idx := range orgFolders {
		orgPath := filepath.Join(mockFS.Root, orgFolders[idx])

		err := fs.WalkDir(
			mockFS.FS, orgPath,
			func(path string, d fs.DirEntry, _ error) error {
				endPath := filepath.Join(tempFolder, dstFolders[idx], strings.TrimPrefix(path, orgPath))
				if d.IsDir() {
					return os.MkdirAll(endPath, os.ModePerm)
				}

				srcFile, err := mockFS.FS.Open(path)
				if err != nil {
					return err
				}

				dstFile, err := os.Create(endPath)
				if err != nil {
					return err
				}
				defer func(f *os.File) {
					_ = f.Close()
				}(dstFile)

				_, err = io.Copy(dstFile, srcFile)
				if err != nil {
					return err
				}

				return nil
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

//nolint:unused
func provideTemporal(lc fx.Lifecycle) (*gnomock.Container, error) {
	options := []gnomock.Option{
		gnomock.WithUseLocalImagesFirst(),
		gnomock.WithEntrypoint(
			"temporal", "server", "start-dev",
			"-n", "business",
			"-n", "subscription",
			"--ip", "0.0.0.0",
		),
		// gnomock.WithLogWriter(os.Stdout),
		gnomock.WithHealthCheck(
			func(_ context.Context, container *gnomock.Container) error {
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
		return nil, err
	}

	lc.Append(
		fx.Hook{
			OnStop: func(_ context.Context) error {
				err = gnomock.Stop(c)
				if err != nil {
					return err
				}

				return nil
			},
		})

	return c, nil
}
