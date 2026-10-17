package helper

func must[T any](v T, err error) T { // want `func must duplicates rkit.Must; delete it and call rkit.Must`
	if err != nil {
		panic(err)
	}

	return v
}

func deref[T any](p *T) T { // want `func deref duplicates rkit.PtrVal`
	if p == nil {
		var zero T

		return zero
	}

	return *p
}

func strVal(p *string) string { // want `func strVal duplicates rkit.PtrVal`
	if p == nil {
		return ""
	}

	return *p
}

type point struct{ x, y int }

func pointVal(p *point) point { // want `func pointVal duplicates rkit.PtrVal`
	if nil == p {
		return point{}
	}

	return *p
}

func contains(s []string, v string) bool { // want `func contains duplicates rkit.Contains`
	for _, x := range s {
		if x == v {
			return true
		}
	}

	return false
}

func firstNonEmpty(vals ...string) string { // want `func firstNonEmpty duplicates rkit.Coalesce`
	for _, v := range vals {
		if v != "" {
			return v
		}
	}

	return ""
}

func coalesce[T comparable](vals ...T) T { // want `func coalesce duplicates rkit.Coalesce`
	var zero T
	for _, v := range vals {
		if v != zero {
			return v
		}
	}

	return zero
}

// Not reported below.

// Must is exported API; removing it is a separate decision.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

func intOrOne(p *int) int {
	if p == nil {
		return 1
	}

	return *p
}

func sliceOrEmpty(p *[]int) []int {
	if p == nil {
		return []int{}
	}

	return *p
}

func containsFold(s []string, v string) bool {
	for _, x := range s {
		if x == v || x == "*" {
			return true
		}
	}

	return false
}

func lastNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return "x"
		}
	}

	return ""
}

type box struct{}

func (box) contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}

	return false
}
