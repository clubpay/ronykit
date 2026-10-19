package mem

import (
	"github.com/clubpay/ronykit/ee/store"
)

// Decoder reads the top-level fields of an encoded record. Numbers should be
// json.Number (or Go numeric types), times RFC 3339 strings or time.Time.
type Decoder func(data []byte) (map[string]any, error)

// JSONDecoder is the default Decoder, matching ee.JSONCodec.
func JSONDecoder(data []byte) (map[string]any, error) {
	return store.DecodeJSON(data)
}

func matchAll(doc map[string]any, conds []store.Cond) bool {
	return store.MatchAll(doc, conds)
}

func compareDoc(a, b any) int {
	return store.CompareValues(a, b)
}
