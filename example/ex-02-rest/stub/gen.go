//go:build ignore

package main

import (
	"github.com/clubpay/ronykit/example/ex-02-rest/api"
	"github.com/clubpay/ronykit/stub/stubgen"
)

func main() {
	stubgen.New(
		stubgen.WithGenFunc(stubgen.GolangStub),
		stubgen.WithPkgName("sampleservice"),
		stubgen.WithFolderName("sammpleservice"),
		stubgen.WithTags("json"),
	).MustGenerate(api.SampleDesc)
}
