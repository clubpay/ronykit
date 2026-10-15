package ee

import "reflect"

func typeOf[T any]() reflect.Type { return reflect.TypeFor[T]() }

func valueOf(v any) reflect.Value { return reflect.ValueOf(v) }
