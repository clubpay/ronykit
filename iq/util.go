package iq

func AsAnyFunc[T, V comparable](fn func(T) V) func(T) any {
	return func(item T) any { return fn(item) }
}
