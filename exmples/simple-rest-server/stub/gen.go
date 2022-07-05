//go:build ignore

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/clubpay/ronykit/exmples/simple-rest-server/api"
	stubgengo "github.com/clubpay/ronykit/stub/gen/go"
	"github.com/clubpay/ronykit/utils"
)

func main() {
	svcDesc := api.NewSample().Desc()
	stubDesc, err := svcDesc.Stub("sampleservicestub", "json")
	if err != nil {
		panic(err)
	}

	rawFile, err := stubgengo.Generate(stubDesc)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(
		fmt.Sprintf("%s.go", strings.ToLower(svcDesc.Name)),
		utils.S2B(rawFile),
		os.ModePerm,
	)
	if err != nil {
		panic(err)
	}
}
