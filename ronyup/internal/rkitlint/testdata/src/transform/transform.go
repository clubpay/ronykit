package transform

import (
	"strconv"
	"strings"
)

type user struct {
	Name   string
	Active bool
}

func names(users []user) []string {
	out := make([]string, 0, len(users)) // want `loop maps one slice to another: use rkit.Map`
	for _, u := range users {
		out = append(out, u.Name)
	}

	return out
}

func trimmed(in []string) []string {
	out := make([]string, len(in)) // want `use rkit.Map`
	for i, s := range in {
		out[i] = strings.TrimSpace(s)
	}

	return out
}

func active(users []user) []user {
	var out []user // want `loop filters a slice: use rkit.Filter`
	for _, u := range users {
		if u.Active {
			out = append(out, u)
		}
	}

	return out
}

func quoted(in []string) []string {
	out := []string{} // want `use rkit.Map`
	for _, s := range in {
		out = append(out, strconv.Quote(s))
	}

	out = append(out, "end")

	return out
}

// Not reported below.

func withIndex(in []string) []string {
	out := make([]string, 0, len(in))
	for i, s := range in {
		out = append(out, strconv.Itoa(i)+s) // want `strconv.Itoa: use rkit.IntToStr`
	}

	return out
}

func selfReference(in []int) []int {
	out := make([]int, 0, len(in))
	for _, v := range in {
		out = append(out, v+len(out))
	}

	return out
}

func copied(in []int) []int {
	out := make([]int, 0, len(in))
	for _, v := range in {
		out = append(out, v)
	}

	return out
}

func twoStatements(in []int) []int {
	out := make([]int, 0, len(in))
	for _, v := range in {
		v *= 2
		out = append(out, v)
	}

	return out
}

func fromMap(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, strings.ToUpper(k))
	}

	return out
}
