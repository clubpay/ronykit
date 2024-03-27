package utils

// Filter returns a new slice containing only the elements in tt for which the match function returns true.
//
//		type User struct {
//			Name string
//			Age  int
//			Active bool
//	}
//
//	var users = []User{
//		{"Tom", 20, true},
//		{"Jack", 22, false},
//		{"Mary", 18, true},
//	}
//
//	var activeUsers = qkit.Filter(func(u User) bool {
//		return u.Active
//	}, users)
//
// fmt.Println(activeUsers)
//
// Playground: https://go.dev/play/p/70YOKRs79OF
func Filter[T any](match func(src T) bool, tt []T) []T {
	ftt := make([]T, 0, len(tt))
	for _, t := range tt {
		if match(t) {
			ftt = append(ftt, t)
		}
	}

	return ftt[:len(ftt):len(ftt)]
}

// Map applies the transformer function to each element of the slice ss.
// The result is a slice of the same length as ss, where the kth element is transformer(ss[k]).
//
//	type User struct {
//		Name string
//		Age  int
//	}
//
//	var users = []User{
//		{"Tom", 20},
//		{"Jack", 22},
//		{"Mary", 18},
//	}
//
//	var names = qkit.Map(func(u User) string {
//		return u.Name
//	}, users)
//
// fmt.Println(names)
//
// Playground: https://go.dev/play/p/wKIa32-rMDn
func Map[S, D any](transformer func(src S) D, ss []S) []D {
	dd := make([]D, len(ss))
	for k, src := range ss {
		dd[k] = transformer(src)
	}

	return dd
}

// Reduce [T, R] reduces the slice tt to a single value r using the reducer
// function. The reducer function takes the current reduced value r and the
// current slice value t and returns a new reduced value.
//
//	type User struct {
//		Name string
//		Age  int
//	}
//
//	var users = []User{
//		{"Tom", 20},
//		{"Jack", 22},
//		{"Mary", 18},
//	}
//
//	var totalAge = qkit.Reduce(func(r int, u User) int {
//		return r + u.Age
//	}, users)
//
// fmt.Println(totalAge)
//
// Playground: https://go.dev/play/p/gf9evzMIMIK
func Reduce[T, R any](reducer func(r R, t T) R, tt []T) R {
	var r R
	for _, t := range tt {
		r = reducer(r, t)
	}

	return r
}

// Paginate will call the given function with start and end indexes
// for a slice of the given size.
//
//	type User struct {
//		Name string
//		Age  int
//	}
//
//	var users = []User{
//		{"Tom", 20},
//		{"Jack", 22},
//		{"Mary", 18},
//		{"Tommy", 20},
//		{"Lin", 22},
//	}
//
//	qkit.Paginate(users, 2, func(start, end int) error {
//		fmt.Println(users[start:end])
//		return nil
//	})
//
// Playground: https://go.dev/play/p/aDiVJEKjgwW
func Paginate[T any](arr []T, pageSize int, fn func(start, end int) error) error {
	start := 0
	for {
		end := start + pageSize
		if end > len(arr) {
			end = len(arr)
		}
		err := fn(start, end)
		if err != nil {
			return err
		}
		start = end
		if start >= len(arr) {
			break
		}
	}

	return nil
}

// MapToArray converts a map's values to a slice.
func MapToArray[K comparable, V any](s map[K]V) []V {
	arr := make([]V, 0, len(s))
	for _, v := range s {
		arr = append(arr, v)
	}

	return arr
}

// MapKeysToArray converts a map's keys to a slice.
func MapKeysToArray[K comparable, V any](s map[K]V) []K {
	arr := make([]K, 0, len(s))
	for k := range s {
		arr = append(arr, k)
	}

	return arr
}

// ArrayToMap converts a slice to a map with the index as the key.
func ArrayToMap[V any](s []V) map[int]V {
	m := make(map[int]V, len(s))
	for idx, v := range s {
		m[idx] = v
	}

	return m
}

func ArrayToSet[T comparable](s []T) map[T]struct{} {
	m := make(map[T]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}

	return m
}

func Contains[T comparable](s []T, v T) bool {
	for _, vv := range s {
		if vv == v {
			return true
		}
	}

	return false
}

func ContainsAny[T comparable](s []T, v []T) bool {
	for _, vv := range v {
		if Contains(s, vv) {
			return true
		}
	}

	return false
}

func ContainsAll[T comparable](s []T, v []T) bool {
	for _, vv := range v {
		if !Contains(s, vv) {
			return false
		}
	}

	return true
}

// First returns the first value found in the map for the given keys.
func First[K, V comparable](in map[K]V, keys ...K) (V, bool) {
	var zero V
	for _, k := range keys {
		if v, ok := in[k]; ok {
			return v, true
		}
	}

	return zero, false
}

func FirstOr[K, V comparable](def V, in map[K]V, keys ...K) V {
	v, ok := First(in, keys...)
	if ok {
		return v
	}

	return def
}

func ForEach[V any](in []V, fn func(*V)) {
	for idx := range in {
		fn(&in[idx])
	}
}

func AddUnique[T comparable](s []T, v T) []T {
	if Contains(s, v) {
		return s
	}

	return append(s, v)
}
