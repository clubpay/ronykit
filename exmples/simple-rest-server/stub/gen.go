//go:build ignore

package main

import (
	"github.com/clubpay/ronykit/exmples/simple-rest-server/api"
	"github.com/clubpay/ronykit/stub/stubgen"
)

func main() {
	stubgen.MustGenerate(
		api.SampleDesc, stubgen.GolangStub,
		"sampleservice",
		"json",
	)
}
