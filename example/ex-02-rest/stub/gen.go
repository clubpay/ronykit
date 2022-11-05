//go:build ignore

package main

import (
	"github.com/clubpay/ronykit/kit/stub/stubgen"
)

func main() {
	stubgen.MustGenerate(
		api.SampleDesc, stubgen.GolangStub,
		"sampleservice",
		"json",
	)
}
