package stdlib

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
)

type code int

func calls(s []string, m map[string]int, n int, n64 int64, u64 uint64, c code) {
	_ = slices.Contains(s, "x")        // want `slices.Contains: use rkit.Contains`
	_ = cmp.Or("", "a")                // want `cmp.Or: use rkit.Coalesce`
	_ = strconv.Itoa(n)                // want `strconv.Itoa: use rkit.IntToStr`
	_ = strconv.FormatInt(n64, 10)     // want `use rkit.Int64ToStr`
	_ = strconv.FormatUint(u64, 10)    // want `use rkit.UInt64ToStr`
	_ = fmt.Sprintf("%d", n)           // want `use rkit.IntToStr`
	_ = fmt.Sprintf("%d", n64)         // want `use rkit.Int64ToStr`
	_ = slices.Collect(maps.Keys(m))   // want `use rkit.MapKeysToArray`
	_ = slices.Collect(maps.Values(m)) // want `use rkit.MapToArray`

	// Not reported:
	_ = strconv.FormatInt(n64, 16)
	_ = fmt.Sprintf("%d", c)
	_ = fmt.Sprintf("%02d", n)
	_ = slices.Index(s, "x")
}
