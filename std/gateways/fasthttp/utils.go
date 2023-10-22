package fasthttp

import (
	"github.com/clubpay/ronykit/kit/utils"
)

type util struct {
	b2s func(b []byte) string
	s2b func(s string) []byte
}

func defaultUtil() util {
	return util{
		b2s: func(b []byte) string {
			return string(b)
		},
		s2b: func(s string) []byte {
			return []byte(s)
		},
	}
}

func speedUtil() util {
	return util{
		b2s: utils.B2S,
		s2b: utils.S2B,
	}
}
