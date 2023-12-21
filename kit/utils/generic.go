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

func Must[T any](src T, err error) T {
	if err != nil {
		panic(err)
	}

	return src
}
