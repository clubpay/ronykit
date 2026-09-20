package rony

import (
	"bytes"
	"testing"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/x/rkit"
)

// TestRkitMatchesKitUtils compares the helpers this module switched from kit/utils to rkit.
func TestRkitMatchesKitUtils(t *testing.T) {
	t.Run("bytes", func(t *testing.T) {
		src := "hello"
		if got, want := rkit.B2S(rkit.S2B(src)), utils.B2S(utils.S2B(src)); got != want {
			t.Fatalf("B2S(S2B) rkit=%q utils=%q", got, want)
		}

		in := []byte("clone")
		if !bytes.Equal(rkit.CloneBytes(in), utils.CloneBytes(in)) {
			t.Fatalf("CloneBytes mismatch: %q vs %q", rkit.CloneBytes(in), utils.CloneBytes(in))
		}
	})

	t.Run("pointers", func(t *testing.T) {
		if rkit.PtrVal[int](nil) != utils.PtrVal[int](nil) {
			t.Fatal("PtrVal(nil) mismatch")
		}
		v := 9
		if rkit.PtrVal(&v) != utils.PtrVal(&v) {
			t.Fatal("PtrVal mismatch")
		}
		if *rkit.ValPtr(5) != *utils.ValPtr(5) {
			t.Fatal("ValPtr mismatch")
		}
		if rkit.TryCast[int]("x") != utils.TryCast[int]("x") {
			t.Fatal("TryCast miss mismatch")
		}
		if rkit.TryCast[string]("ok") != utils.TryCast[string]("ok") {
			t.Fatal("TryCast hit mismatch")
		}
		if rkit.Coalesce("", "a") != utils.Coalesce("", "a") {
			t.Fatal("Coalesce mismatch")
		}
		if rkit.Must(1, nil) != utils.Must(1, nil) {
			t.Fatal("Must mismatch")
		}
		if rkit.Ok("x", nil) != utils.Ok("x", nil) {
			t.Fatal("Ok mismatch")
		}
	})

	t.Run("numbers", func(t *testing.T) {
		cases := []string{"0", "12", "1.25", "-3", "nope"}
		for _, s := range cases {
			if rkit.StrToInt(s) != utils.StrToInt(s) {
				t.Fatalf("StrToInt(%q) rkit=%d utils=%d", s, rkit.StrToInt(s), utils.StrToInt(s))
			}
			if rkit.StrToInt32(s) != utils.StrToInt32(s) {
				t.Fatalf("StrToInt32(%q) mismatch", s)
			}
			if rkit.StrToInt64(s) != utils.StrToInt64(s) {
				t.Fatalf("StrToInt64(%q) mismatch", s)
			}
			if rkit.StrToUInt(s) != utils.StrToUInt(s) {
				t.Fatalf("StrToUInt(%q) mismatch", s)
			}
			if rkit.StrToUInt32(s) != utils.StrToUInt32(s) {
				t.Fatalf("StrToUInt32(%q) mismatch", s)
			}
			if rkit.StrToUInt64(s) != utils.StrToUInt64(s) {
				t.Fatalf("StrToUInt64(%q) mismatch", s)
			}
			if rkit.StrToFloat32(s) != utils.StrToFloat32(s) {
				t.Fatalf("StrToFloat32(%q) mismatch", s)
			}
			if rkit.StrToFloat64(s) != utils.StrToFloat64(s) {
				t.Fatalf("StrToFloat64(%q) mismatch", s)
			}
		}

		if rkit.IntToStr(42) != utils.IntToStr(42) {
			t.Fatal("IntToStr mismatch")
		}
		if rkit.Int64ToStr(42) != utils.Int64ToStr(42) {
			t.Fatal("Int64ToStr mismatch")
		}
		if rkit.Float64ToStr(1.5) != utils.Float64ToStr(1.5) {
			t.Fatal("Float64ToStr mismatch")
		}
	})

	t.Run("collections", func(t *testing.T) {
		in := []int{1, 2, 3, 2}
		rFiltered := rkit.Filter(in, func(v int) bool { return v%2 == 0 })
		uFiltered := utils.Filter(func(v int) bool { return v%2 == 0 }, in)
		if len(rFiltered) != len(uFiltered) || rFiltered[0] != uFiltered[0] {
			t.Fatalf("Filter mismatch: %v vs %v", rFiltered, uFiltered)
		}

		rMapped := rkit.Map(in, func(v int) string { return rkit.IntToStr(v) })
		uMapped := utils.Map(func(v int) string { return utils.IntToStr(v) }, in)
		if len(rMapped) != len(uMapped) {
			t.Fatalf("Map len mismatch: %v vs %v", rMapped, uMapped)
		}
		for i := range rMapped {
			if rMapped[i] != uMapped[i] {
				t.Fatalf("Map[%d] mismatch: %q vs %q", i, rMapped[i], uMapped[i])
			}
		}

		src := map[string]int{"a": 1, "b": 2}
		if len(rkit.MapToArray(src)) != len(utils.MapToArray(src)) {
			t.Fatal("MapToArray len mismatch")
		}
		if len(rkit.MapKeysToArray(src)) != len(utils.MapKeysToArray(src)) {
			t.Fatal("MapKeysToArray len mismatch")
		}

		rUnique := rkit.AddUnique([]string{"a"}, "a")
		uUnique := utils.AddUnique([]string{"a"}, "a")
		if len(rUnique) != len(uUnique) {
			t.Fatalf("AddUnique existing mismatch: %v vs %v", rUnique, uUnique)
		}
		rUnique = rkit.AddUnique([]string{"a"}, "b")
		uUnique = utils.AddUnique([]string{"a"}, "b")
		if len(rUnique) != len(uUnique) || rUnique[1] != uUnique[1] {
			t.Fatalf("AddUnique new mismatch: %v vs %v", rUnique, uUnique)
		}
	})

	t.Run("case", func(t *testing.T) {
		// Inputs we actually pass after the migration (handler names, stub names).
		// rkit.ToCamel also treats / | \ as separators; kit/utils does not.
		inputs := []string{"", "echo", "EchoHandler", "handle_request", "user-id", "user.id", "ID"}
		for _, in := range inputs {
			if got, want := rkit.ToCamel(in), utils.ToCamel(in); got != want {
				t.Fatalf("ToCamel(%q) rkit=%q utils=%q", in, got, want)
			}
			if got, want := rkit.ToLowerCamel(in), utils.ToLowerCamel(in); got != want {
				t.Fatalf("ToLowerCamel(%q) rkit=%q utils=%q", in, got, want)
			}
			if got, want := rkit.ToScreamingSnake(in), utils.ToScreamingSnake(in); got != want {
				t.Fatalf("ToScreamingSnake(%q) rkit=%q utils=%q", in, got, want)
			}
		}
	})

	t.Run("time", func(t *testing.T) {
		r := rkit.TimeUnix()
		u := utils.TimeUnix()
		if r-u > 1 || u-r > 1 {
			t.Fatalf("TimeUnix drifted: rkit=%d utils=%d", r, u)
		}
		if rkit.NanoTime() <= 0 || utils.NanoTime() <= 0 {
			t.Fatal("NanoTime must be positive")
		}
	})
}
