package utils

import "encoding/json"

// PtrVal returns the value of the pointer src. It is a dereference operation.
func PtrVal[T any](src *T) T {
	if src == nil {
		var dst T

		return dst
	}

	return *src
}

// ValPtr returns the pointer of the src. It is a reference operation.
func ValPtr[T any](src T) *T {
	return &src
}

// ValPtrOrNil returns the pointer of the src if src is not zero value, otherwise nil.
func ValPtrOrNil[T comparable](src T) *T {
	var zero T
	if src == zero {
		return nil
	}

	return &src
}

// Must panics if err is not nil
func Must[T any](src T, err error) T {
	if err != nil {
		panic(err)
	}

	return src
}

// Ok returns the value and ignores the error
func Ok[T any](v T, _ error) T {
	return v
}

// OkOr returns the value if err is nil, otherwise returns the fallback value
func OkOr[T any](v T, err error, fallback T) T {
	if err != nil {
		return fallback
	}

	return v
}

// TryCast tries to cast the input value to the target type. If the cast fails, it
// returns the zero value of the target type.
func TryCast[T any](v any) T {
	t, _ := v.(T)

	return t
}

// Coalesce returns the first non-zero value from the input list.
func Coalesce[T comparable](in ...T) T {
	var zero T
	for _, v := range in {
		if v != zero {
			return v
		}
	}

	return zero
}

// ToMap Converts any type to a map[string]interface{}.
func ToMap(s any) map[string]any {
	m := make(map[string]any)
	_ = json.Unmarshal(Ok(json.Marshal(s)), &m)

	return m
}
