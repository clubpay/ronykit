package utils

func PtrVal[T any](src *T) T {
	if src == nil {
		var dst T

		return dst
	}

	return *src
}

func ValPtr[T any](src T) *T {
	return &src
}

func ValPtrOrNil[T comparable](src T) *T {
	var zero T
	if src == zero {
		return nil
	}

	return &src
}
