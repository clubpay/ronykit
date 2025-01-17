package testenv

import (
	"context"
	"testing"

	"github.com/clubpay/ronykit/flow"
	"github.com/clubpay/ronykit/kit/utils"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFlow(t *testing.T) {
	Convey("Flow", t, func(c C) {
		sdk, err := flow.NewSDK(
			flow.Config{
				Namespace: "kitTest",
				HostPort:  temporalHostPort,
			},
		)
		c.So(err, ShouldBeNil)

		WFSelect.Init(sdk, "hello")

		ctx := context.Background()
		wr, err := WFSelect.Execute(ctx, "Req1", flow.ExecuteWorkflowOptions{})
		c.So(err, ShouldBeNil)

		res, err := wr.Get(ctx)
		c.So(err, ShouldBeNil)
		c.So(*res, ShouldEqual, "hi")

	})
}

var WFSelect = flow.NewWorkflow(
	"Select",
	func(initArg string) flow.WorkflowFunc[string, string] {
		return func(ctx *flow.WorkflowContext[string, string], req string) (*string, error) {
			return utils.ValPtr("hi"), nil
		}
	},
)
