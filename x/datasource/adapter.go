package datasource

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

type DBParams struct {
	Host       string
	Port       int
	User       string
	Pass       string
	DB         string
	SSLMode    string
	Migrations fs.FS
}

func (params DBParams) DSN() string {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s database=%s port=%d sslmode=disable",
		params.Host, params.User, params.Pass, params.DB, params.Port,
	)

	if params.SSLMode != "" {
		dsn += fmt.Sprintf(" sslmode=%s", params.SSLMode)
	}

	return dsn
}

func InitDB(in, out string) fx.Option {
	var annotations []fx.Annotation
	if in != "" {
		annotations = append(annotations, fx.ParamTags(fmt.Sprintf("name:%q", in)))
	}

	if out != "" {
		annotations = append(annotations, fx.ResultTags(fmt.Sprintf("name:%q", out)))
	}

	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(params DBParams) (*sql.DB, error) {
					db, err := sql.Open("pgx", params.DSN())
					if err != nil {
						return nil, errors.Wrap(err, "failed to open db")
					}

					db.SetMaxOpenConns(12)
					db.SetConnMaxLifetime(time.Minute)
					db.SetMaxIdleConns(5)

					err = runMigrations(db, params)
					if err != nil {
						return nil, errors.Wrap(err, "failed to run migrations")
					}

					err = db.Ping()
					if err != nil {
						return nil, errors.Wrap(err, "failed to ping db")
					}

					return db, nil
				},
				annotations...,
			),
		),
	)
}

func runMigrations(db *sql.DB, params DBParams) error {
	if params.Migrations == nil {
		return nil
	}

	drv, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: "migrate",
		DatabaseName:    params.DB,
		SchemaName:      "public",
	})
	if err != nil {
		return err
	}

	src, err := httpfs.New(http.FS(params.Migrations), ".")
	if err != nil {
		return err
	}
	defer func(src source.Driver) { _ = src.Close() }(src)

	m, err := migrate.NewWithInstance("migration", src, params.DB, drv)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

type RedisParams struct {
	Host               string
	Port               int
	User               string
	Pass               string
	DBNumber           int
	InsecureSkipVerify bool
}

func (params RedisParams) DSN() string {
	if params.DBNumber > 0 {
		return fmt.Sprintf(
			"redis://%s:%s@%s:%d/%d",
			params.User, url.QueryEscape(params.Pass), params.Host, params.Port, params.DBNumber,
		)
	} else {
		return fmt.Sprintf(
			"redis://%s:%s@%s:%d",
			params.User, url.QueryEscape(params.Pass), params.Host, params.Port,
		)
	}
}

func InitRedis(in, out string) fx.Option {
	var annotations []fx.Annotation
	if in != "" {
		annotations = append(annotations, fx.ParamTags(fmt.Sprintf("name:%q", in)))
	}

	if out != "" {
		annotations = append(annotations, fx.ResultTags(fmt.Sprintf("name:%q", out)))
	}

	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(params RedisParams) (*redis.Client, error) {
					opt, err := redis.ParseURL(params.DSN())
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
				annotations...,
			),
		),
	)
}

type S3Params struct {
	URL       string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

func InitS3(in, out string) fx.Option {
	var annotations []fx.Annotation
	if in != "" {
		annotations = append(annotations, fx.ParamTags(fmt.Sprintf("name:%q", in)))
	}

	if out != "" {
		annotations = append(annotations, fx.ResultTags(fmt.Sprintf("name:%q", out)))
	}

	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(params S3Params) (*minio.Client, error) {
					// Initialize minio client object.
					return minio.New(params.URL, &minio.Options{
						Creds:  credentials.NewStaticV4(params.AccessKey, params.SecretKey, ""),
						Secure: params.UseSSL,
					})
				},
				annotations...,
			),
		),
	)
}

type MinioParams struct {
	URL       string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

func InitMinioClient(in, out string) fx.Option {
	var annotations []fx.Annotation
	if in != "" {
		annotations = append(annotations, fx.ParamTags(fmt.Sprintf("name:%q", in)))
	}

	if out != "" {
		annotations = append(annotations, fx.ResultTags(fmt.Sprintf("name:%q", out)))
	}

	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(params MinioParams) (*minio.Client, error) {
					// Initialize minio client object.
					return minio.New(params.URL, &minio.Options{
						Creds:  credentials.NewStaticV4(params.AccessKey, params.SecretKey, ""),
						Secure: params.UseSSL,
					})
				},
				annotations...,
			),
		),
	)
}

func InitMinioCore(in, out string) fx.Option {
	var annotations []fx.Annotation
	if in != "" {
		annotations = append(annotations, fx.ParamTags(fmt.Sprintf("name:%q", in)))
	}

	if out != "" {
		annotations = append(annotations, fx.ResultTags(fmt.Sprintf("name:%q", out)))
	}

	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(params MinioParams) (*minio.Core, error) {
					// Initialize minio client object.
					return minio.NewCore(params.URL, &minio.Options{
						Creds:  credentials.NewStaticV4(params.AccessKey, params.SecretKey, ""),
						Secure: params.UseSSL,
					})
				},
				annotations...,
			),
		),
	)
}
