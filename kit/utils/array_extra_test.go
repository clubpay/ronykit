package utils_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/clubpay/ronykit/kit/utils"
)

func TestArrayHelpers(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	evens := utils.Filter(func(v int) bool { return v%2 == 0 }, items)
	if !reflect.DeepEqual(evens, []int{2, 4}) {
		t.Fatalf("unexpected filter result: %v", evens)
	}

	squares := utils.Map(func(v int) int { return v * v }, items)
	if !reflect.DeepEqual(squares, []int{1, 4, 9, 16, 25}) {
		t.Fatalf("unexpected map result: %v", squares)
	}

	sum := utils.Reduce(func(r int, v int) int { return r + v }, items)
	if sum != 15 {
		t.Fatalf("unexpected reduce result: %d", sum)
	}

	var gotPages [][2]int
	err := utils.Paginate(items, 2, func(start, end int) error {
		gotPages = append(gotPages, [2]int{start, end})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	wantPages := [][2]int{{0, 2}, {2, 4}, {4, 5}}
	if !reflect.DeepEqual(gotPages, wantPages) {
		t.Fatalf("unexpected paginate result: %v", gotPages)
	}

	expectedErr := errors.New("stop")
	err = utils.Paginate(items, 2, func(start, end int) error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected paginate error, got %v", err)
	}

	values := map[string]int{"a": 1, "b": 2}
	arr := utils.MapToArray(values)
	if len(arr) != 2 {
		t.Fatalf("unexpected map to array length: %d", len(arr))
	}

	keys := utils.MapKeysToArray(values)
	if len(keys) != 2 {
		t.Fatalf("unexpected map keys length: %d", len(keys))
	}

	mapped := utils.MapToArrayFunc(values, func(k string, v int) string { return k + utils.IntToStr(v) })
	if len(mapped) != 2 {
		t.Fatalf("unexpected map to array func length: %d", len(mapped))
	}

	m := utils.ArrayToMap([]string{"a", "b"})
	if m[0] != "a" || m[1] != "b" {
		t.Fatalf("unexpected array to map: %v", m)
	}

	set := utils.ArrayToSet([]string{"a", "a", "b"})
	if _, ok := set["a"]; !ok || len(set) != 2 {
		t.Fatalf("unexpected array to set: %v", set)
	}

	if !utils.Contains(items, 3) {
		t.Fatal("expected contains to find 3")
	}
	if !utils.ContainsAny(items, []int{9, 4}) {
		t.Fatal("expected contains any to find 4")
	}
	if !utils.ContainsAll(items, []int{2, 5}) {
		t.Fatal("expected contains all to find 2 and 5")
	}

	first, ok := utils.First(map[string]string{"a": "x", "b": "y"}, "c", "b", "a")
	if !ok || first != "y" {
		t.Fatalf("unexpected first result: %v, %v", first, ok)
	}

	if v := utils.FirstOr("def", map[string]string{"a": "x"}, "b"); v != "def" {
		t.Fatalf("unexpected first or result: %v", v)
	}

	mutable := []int{1, 2, 3}
	utils.ForEach(mutable, func(v *int) { *v *= 2 })
	if !reflect.DeepEqual(mutable, []int{2, 4, 6}) {
		t.Fatalf("unexpected foreach result: %v", mutable)
	}

	unique := utils.AddUnique([]int{1, 2}, 2)
	if !reflect.DeepEqual(unique, []int{1, 2}) {
		t.Fatalf("unexpected add unique result: %v", unique)
	}
	unique = utils.AddUnique([]int{1, 2}, 3)
	if !reflect.DeepEqual(unique, []int{1, 2, 3}) {
		t.Fatalf("unexpected add unique result: %v", unique)
	}
}
