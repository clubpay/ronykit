package testenv

import (
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
				HostPort:  "127.0.0.1:7333",
			},
		)
		c.So(err, ShouldBeNil)

		WFSelect.Init(sdk, "hello")

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
