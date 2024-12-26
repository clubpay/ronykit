package iq

import "reflect"

// Iterator is an alias for a function to iterate over data.
type Iterator[T comparable] func() (item T, ok bool)

// Query is the type returned from query functions. It can be iterated manually
// as shown in the example.
type Query[T comparable] struct {
	Iterate func() Iterator[T]
}

// KeyValue is a type that is used to iterate over a map (if query is created
// from a map). This type is also used by ToMap() method to output result of a
// query into a map.
type KeyValue[K, V comparable] struct {
	Key   K
	Value V
}

// Iterable is an interface that has to be implemented by a custom collection
// to work with iq.
type Iterable[T comparable] interface {
	Iterate() Iterator[T]
}

func From[T comparable](source Iterable[T]) Query[T] {
	return FromIterable[T](source)
}

func FromSlice[T comparable](source []T) Query[T] {
	ln := len(source)

	return Query[T]{
		Iterate: func() Iterator[T] {
			index := 0

			return func() (item T, ok bool) {
				ok = index < ln
				if ok {
					item = source[index]
					index++
				}

				return
			}
		},
	}
}

func FromMap[K, V comparable](source map[K]V) Query[KeyValue[K, V]] {
	src := reflect.ValueOf(source)
	ln := src.Len()

	return Query[KeyValue[K, V]]{
		Iterate: func() Iterator[KeyValue[K, V]] {
			index := 0
			keys := src.MapKeys()

			return func() (item KeyValue[K, V], ok bool) {
				ok = index < ln
				if ok {
					key := keys[index]
					item = KeyValue[K, V]{
						Key:   key.Interface().(K),               //nolint:forcetypeassert
						Value: src.MapIndex(key).Interface().(V), //nolint:forcetypeassert
					}

					index++
				}

				return
			}
		},
	}
}

// FromChannel initializes a query with a passed channel, iq iterates over
// the channel until it is closed.
func FromChannel[T comparable](source <-chan T) Query[T] {
	return Query[T]{
		Iterate: func() Iterator[T] {
			return func() (item T, ok bool) {
				item, ok = <-source

				return
			}
		},
	}
}

// FromString initializes a iq query with passed string, iq iterates over
// runes of string.
func FromString(source string) Query[rune] {
	runes := []rune(source)
	n := len(runes)

	return Query[rune]{
		Iterate: func() Iterator[rune] {
			index := 0

			return func() (item rune, ok bool) {
				ok = index < n
				if ok {
					item = runes[index]
					index++
				}

				return
			}
		},
	}
}

// FromIterable initializes a iq query with custom collection passed. This
// collection has to implement Iterable interface, iq iterates over items,
// that has to implement Comparable interface or be basic types.
func FromIterable[T comparable](source Iterable[T]) Query[T] {
	return Query[T]{
		Iterate: source.Iterate,
	}
}

// Range generates a sequence of integral numbers within a specified range.
func Range[T Ordered](start T, count int) Query[T] {
	return Query[T]{
		Iterate: func() Iterator[T] {
			index := 0
			current := start

			return func() (item T, ok bool) {
				if index >= count {
					return Zero[T](), false
				}

				item, ok = current, true

				index++
				current++

				return
			}
		},
	}
}

// Repeat generates a sequence that contains one repeated value.
func Repeat[T comparable](value T, count int) Query[T] {
	return Query[T]{
		Iterate: func() Iterator[T] {
			index := 0

			return func() (item T, ok bool) {
				if index >= count {
					return Zero[T](), false
				}

				item, ok = value, true
				index++

				return
			}
		},
	}
}

func Zero[T any]() T {
	var zero T

	return zero
}

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}
