package fixmod

import (
	"strconv"

	rk "github.com/clubpay/ronykit/x/rkit"
)

func aliased() int {
	n, err := strconv.Atoi(rk.IntToStr(1))
	if err != nil {
		panic(err)
	}

	return n
}
