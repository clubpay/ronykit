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

// Map applies the transformer function to each element of the slice ss. The result is a slice of the same length as ss, where the kth element is transformer(ss[k]).
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

func MapToArray[K comparable, V any](s map[K]V) []V {
	arr := make([]V, 0, len(s))
	for _, v := range s {
		arr = append(arr, v)
	}

	return arr
}
