package utils

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
