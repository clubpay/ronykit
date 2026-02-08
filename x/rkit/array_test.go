package rkit

import (
	"errors"
	"testing"
)

func TestArrayTransforms(t *testing.T) {
	input := []int{1, 2, 3, 4}

	filtered := Filter(input, func(v int) bool { return v%2 == 0 })
	if len(filtered) != 2 || filtered[0] != 2 || filtered[1] != 4 {
		t.Fatalf("Filter = %v, want [2 4]", filtered)
	}

	mapped := Map(input, func(v int) string { return IntToStr(v) })
	if len(mapped) != 4 || mapped[0] != "1" || mapped[3] != "4" {
		t.Fatalf("Map = %v, want [1 2 3 4]", mapped)
	}

	sum := Reduce(func(r int, v int) int { return r + v }, input)
	if sum != 10 {
		t.Fatalf("Reduce sum = %d, want 10", sum)
	}
}

func TestPaginate(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	collected := make([]int, 0, len(input))

	err := Paginate(input, 2, func(start, end int) error {
		collected = append(collected, input[start:end]...)
		return nil
	})
	if err != nil {
		t.Fatalf("Paginate err = %v, want nil", err)
	}
	if len(collected) != len(input) {
		t.Fatalf("Paginate collected len = %d, want %d", len(collected), len(input))
	}
	for i, v := range collected {
		if v != input[i] {
			t.Fatalf("Paginate collected[%d] = %d, want %d", i, v, input[i])
		}
	}

	sentinel := errors.New("stop")
	err = Paginate(input, 2, func(start, end int) error {
		return sentinel
	})
	if err != sentinel {
		t.Fatalf("Paginate err = %v, want %v", err, sentinel)
	}
}

func TestMapConversions(t *testing.T) {
	src := map[string]int{"a": 1, "b": 2}

	values := MapToArray(src)
	if len(values) != 2 {
		t.Fatalf("MapToArray len = %d, want 2", len(values))
	}

	valueSet := map[int]struct{}{}
	for _, v := range values {
		valueSet[v] = struct{}{}
	}
	if _, ok := valueSet[1]; !ok {
		t.Fatalf("MapToArray missing value 1")
	}
	if _, ok := valueSet[2]; !ok {
		t.Fatalf("MapToArray missing value 2")
	}

	keyValues := MapToArrayFunc(src, func(k string, v int) string {
		return k + IntToStr(v)
	})
	if len(keyValues) != 2 {
		t.Fatalf("MapToArrayFunc len = %d, want 2", len(keyValues))
	}

	keys := MapKeysToArray(src)
	if len(keys) != 2 {
		t.Fatalf("MapKeysToArray len = %d, want 2", len(keys))
	}
}

func TestArrayToMapAndContains(t *testing.T) {
	input := []string{"a", "bb"}
	m := ArrayToMapFunc(input, func(v string) int { return len(v) })
	if got := m[1]; got != "a" {
		t.Fatalf("ArrayToMapFunc[1] = %q, want %q", got, "a")
	}
	if got := m[2]; got != "bb" {
		t.Fatalf("ArrayToMapFunc[2] = %q, want %q", got, "bb")
	}

	if !Contains(input, "a") {
		t.Fatalf("Contains = false, want true")
	}
	if Contains(input, "c") {
		t.Fatalf("Contains = true, want false")
	}
}
