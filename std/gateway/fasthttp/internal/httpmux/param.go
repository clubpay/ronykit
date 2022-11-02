package httpmux

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
)

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the httpMux.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) string {
	for _, p := range ps {
		if p.Key == name {
			return string(utils.S2B(p.Value))
		}
	}

	return ""
}

type DecoderFunc func(bag Params, data []byte) (kit.Message, error)
