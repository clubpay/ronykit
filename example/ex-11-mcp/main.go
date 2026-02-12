package main

import (
	"context"
	"fmt"
	"os"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/mcp"
)

func main() {
	defer kit.NewServer(
		kit.WithGateway(
			mcp.MustNew(
				mcp.WithName("SampleName"),
				mcp.WithInstructions("This is a sample server just says hi"),
			),
		),
		kit.WithServiceBuilder(
			desc.NewService("SampleName").
				AddContract(
					desc.NewContract().
						In(&SayHiInput{}).
						Out(&SayHiOutput{}).
						AddRoute(
							desc.Route("sayHi", mcp.Selector{
								Name:        "SayHi",
								Title:       "Say Hi",
								Description: "It just says hi",
							}),
						).
						SetHandler(sayHi),
				),
		),
	).
		Start(context.Background()).
		PrintRoutesCompact(os.Stdout).
		Shutdown(context.Background(), os.Interrupt)
}

type SayHiInput struct {
	Name  string `json:"name" jsonschema:"required"`
	Phone string `json:"phone" jsonschema:"required,format:phone"`
}

type SayHiOutput struct {
	Msg string `json:"msg" jsonschema:"required"`
}

func sayHi(ctx *kit.Context) {
	in := ctx.In().GetMsg().(*SayHiInput)
	fmt.Println("sayHi called with name:", in.Name)

	ctx.In().Reply().SetMsg(&SayHiOutput{Msg: "Hi from Ronykit " + in.Name}).Send()
}
