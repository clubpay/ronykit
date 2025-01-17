package flow_test

import (
	"fmt"
	"testing"

	"github.com/clubpay/ronykit/flow"
	. "github.com/smartystreets/goconvey/convey"
)

func TestWorkflowFilters(t *testing.T) {
	Convey("Testing WorkflowFilters", t, func(c C) {
		c.So(
			flow.AND(
				flow.GT(flow.WorkflowFilterTemporalScheduledById, "10"),
				flow.EQ(flow.WorkflowFilterExecutionStatus, flow.ExecutionStatusRunning),
			),
			ShouldEqual,
			fmt.Sprintf("%s > '10' AND %s = '%s'",
				flow.WorkflowFilterTemporalScheduledById,
				flow.WorkflowFilterExecutionStatus,
				flow.ExecutionStatusRunning,
			),
		)

		c.So(
			flow.AND(
				flow.StartsWith(flow.WorkflowFilterTemporalScheduledById, "10"),
				flow.AND(
					flow.GT(flow.WorkflowFilterTemporalScheduledById, "10"),
					flow.EQ(flow.WorkflowFilterExecutionStatus, flow.ExecutionStatusRunning),
				),
			),
			ShouldEqual,
			fmt.Sprintf("%s STARTS_WITH '10' AND (%s > '10' AND %s = '%s')",
				flow.WorkflowFilterTemporalScheduledById,
				flow.WorkflowFilterTemporalScheduledById,
				flow.WorkflowFilterExecutionStatus,
				flow.ExecutionStatusRunning,
			),
		)
	})
}
