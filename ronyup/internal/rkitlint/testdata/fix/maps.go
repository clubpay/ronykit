package fixmod

import "sort"

func keys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}

func sortedValues(m map[string]int) []int {
	out := []int{}
	for _, v := range m {
		out = append(out, v)
	}

	sort.Ints(out)

	return out
}
