package store

import "testing"

func TestCompareValuesBool(t *testing.T) {
	cases := []struct {
		a, b any
		want int
	}{
		{true, true, 0},
		{false, false, 0},
		{false, true, -1},
		{true, false, 1},
	}

	for _, c := range cases {
		if got := CompareValues(c.a, c.b); got != c.want {
			t.Errorf("CompareValues(%v, %v) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
