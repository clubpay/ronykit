package di

import (
	"io/fs"
	"reflect"

	"github.com/clubpay/ronykit/x/datasource"
	"github.com/clubpay/ronykit/x/rkit"
	"go.uber.org/fx"
)

func ProvideDBParams[Settings any](migrationFS fs.FS) fx.Option {
	return fx.Provide(
		func(set *Settings) datasource.DBParams {
			return datasource.DBParams{
				Host:       getField[string](set, "DB", "Host"),
				Port:       getField[int](set, "DB", "Port"),
				User:       getField[string](set, "DB", "User"),
				Pass:       getField[string](set, "DB", "Pass"),
				DB:         getField[string](set, "DB", "DB"),
				Migrations: rkit.Must(fs.Sub(migrationFS, "internal/repo/v0/data/db/migrations")),
			}
		},
	)
}

func ProvideRedisParams[Settings any]() fx.Option {
	return fx.Provide(
		func(set *Settings) datasource.RedisParams {
			return datasource.RedisParams{
				Host: getField[string](set, "Redis", "Host"),
				Port: getField[int](set, "Redis", "Port"),
				User: getField[string](set, "Redis", "User"),
				Pass: getField[string](set, "Redis", "Pass"),
			}
		},
	)
}

func getField[Result, Settings any](settings Settings, parentField, childField string) Result {
	return reflect.Indirect(reflect.ValueOf(settings)). //nolint:forcetypeassert
								FieldByName(parentField).
								FieldByName(childField).
								Interface().(Result)
}
