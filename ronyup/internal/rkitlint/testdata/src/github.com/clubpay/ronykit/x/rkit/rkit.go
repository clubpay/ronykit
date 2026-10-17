// Package rkit stands in for x/rkit: the analyzer must not report rkit's own implementations.
package rkit

func Must[T any](src T, err error) T {
	if err != nil {
		panic(err)
	}

	return src
}

func must[T any](src T, err error) T {
	if err != nil {
		panic(err)
	}

	return src
}
