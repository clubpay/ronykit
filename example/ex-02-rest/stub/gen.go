////go:build ignore

package main

import (
	"github.com/clubpay/ronykit/example/ex-02-rest/api"
	"github.com/clubpay/ronykit/stub/stubgen"
)

func main() {
	stubgen.New(
		stubgen.WithGenEngine(stubgen.NewGolangEngine(stubgen.GolangConfig{
			PkgName: "sampleservice",
		})),
		stubgen.WithFolderName("sampleservice"),
		stubgen.WithTags("json"),
	).MustGenerate(api.SampleDesc)
}
