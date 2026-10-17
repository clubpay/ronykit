package mapcollect

import "sort"

func keys(m map[string]int) []string {
	out := make([]string, 0, len(m)) // want `loop collects map keys: use rkit.MapKeysToArray`
	for k := range m {
		out = append(out, k)
	}

	return out
}

func values(m map[string]int) []int {
	out := []int{} // want `loop collects map values: use rkit.MapToArray`
	for _, v := range m {
		out = append(out, v)
	}

	return out
}

func keysThenSort(m map[string]bool) []string {
	out := make([]string, 0) // want `loop collects map keys`
	for k := range m {
		out = append(out, k)
	}

	sort.Strings(out)

	return out
}

func nilSlice(m map[string]int) []string {
	var out []string // want `loop collects map keys`
	for k := range m {
		out = append(out, k)
	}

	return out
}

type names []string

func named(m map[string]int) names {
	out := make(names, 0) // want `loop collects map keys`
	for k := range m {
		out = append(out, k)
	}

	return out
}

// Not reported below.

func transformed(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k+"x")
	}

	return out
}

func filtered(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		if v > 0 {
			out = append(out, k)
		}
	}

	return out
}

func fromSlice(s []string) []string {
	out := make([]string, 0, len(s))
	for _, v := range s {
		out = append(out, v)
	}

	return out
}

func widened(m map[string]int) []any {
	out := make([]any, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}

func prefilled(m map[string]int) []string {
	out := []string{"first"}
	for k := range m {
		out = append(out, k)
	}

	return out
}

func sideEffectCap(m map[string]int, next func() int) []string {
	out := make([]string, 0, next())
	for k := range m {
		out = append(out, k)
	}

	return out
}
