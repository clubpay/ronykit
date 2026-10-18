package discard

import (
	"encoding/json"
	"os"
	"strconv"
)

func dropped(v any, s string) {
	n, _ := strconv.Atoi(s)              // want `error dropped: use rkit.StrToInt`
	i, _ := strconv.ParseInt(s, 10, 64)  // want `error dropped: use rkit.StrToInt64`
	u, _ := strconv.ParseUint(s, 10, 64) // want `error dropped with _: use rkit.Ok`
	x, _ := strconv.ParseFloat(s, 64)    // want `error dropped: use rkit.StrToFloat64`
	b, _ := json.Marshal(v)              // want `error dropped: use rkit.ToJSON`
	i32, _ := strconv.ParseInt(s, 10, 32) // want `error dropped with _: use rkit.Ok`
	wd, _ := os.Getwd()                  // want `error dropped with _: use rkit.Ok`
	str, _ := v.(string)                 // want `type assertion with the ok result dropped: use rkit.TryCast`

	var f *os.File
	f, _ = v.(*os.File) // want `use rkit.TryCast`

	_, _, _, _, _, _, _, _, _ = n, i, u, x, b, i32, wd, str, f
}

// Not reported below.

func kept(m map[string]int, ch chan int, s string) {
	v, _ := m["k"]
	r, _ := <-ch
	_, err := strconv.Atoi(s)
	_, _ = strconv.Atoi(s)

	_, _, _ = v, r, err
}
