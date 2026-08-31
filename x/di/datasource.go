package di

import (
	"io/fs"
	"reflect"

	"github.com/clubpay/ronykit/x/datasource"

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
				Migrations: migrationFS,
			}
		},
	)
}

func ProvideRedisParams[Settings any]() fx.Option {
	return fx.Provide(
		func(set *Settings) datasource.RedisParams {
			return datasource.RedisParams{
				Host:               getField[string](set, "Redis", "Host"),
				Port:               getField[int](set, "Redis", "Port"),
				User:               getField[string](set, "Redis", "User"),
				Pass:               getField[string](set, "Redis", "Pass"),
				TLS:                getOptionalField[bool](set, "Redis", "TLS"),
				InsecureSkipVerify: getOptionalField[bool](set, "Redis", "InsecureSkipVerify"),
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

// getOptionalField behaves like getField but returns the zero value of Result
// instead of panicking when parentField or childField is absent from settings,
// so existing settings structs that predate a new optional field keep working.
func getOptionalField[Result, Settings any](settings Settings, parentField, childField string) Result {
	var zero Result

	parent := reflect.Indirect(reflect.ValueOf(settings)).FieldByName(parentField)
	if !parent.IsValid() {
		return zero
	}

	field := parent.FieldByName(childField)
	if !field.IsValid() {
		return zero
	}

	result, ok := reflect.TypeAssert[Result](field)
	if !ok {
		return zero
	}

	return result
}
