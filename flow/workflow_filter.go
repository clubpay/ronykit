package flow

import (
	"fmt"
	"strings"

	"github.com/clubpay/ronykit/kit/utils"
)

type WorkflowFilterName string

const (
	WorkflowFilterWorkflowId            WorkflowFilterName = "WorkflowId"
	WorkflowFilterWorkflowType          WorkflowFilterName = "WorkflowType"
	WorkflowFilterTemporalScheduledById WorkflowFilterName = "TemporalScheduledById"
	WorkflowFilterTaskQueue             WorkflowFilterName = "TaskQueue"
	WorkflowFilterExecutionStatus       WorkflowFilterName = "ExecutionStatus"

	ExecutionStatusCompleted      = "Completed"
	ExecutionStatusFailed         = "Failed"
	ExecutionStatusRunning        = "Running"
	ExecutionStatusTerminated     = "Terminated"
	ExecutionStatusCanceled       = "Canceled"
	ExecutionStatusTimedOut       = "TimedOut"
	ExecutionStatusContinuedAsNew = "ContinuedAsNew"
)

func AND(a, b string) string {
	if strings.Contains(a, "AND") || strings.Contains(a, "OR") {
		a = fmt.Sprintf("(%s)", a)
	}

	if strings.Contains(b, "AND") || strings.Contains(b, "OR") {
		b = fmt.Sprintf("(%s)", b)
	}

	return fmt.Sprintf(`%s AND %s`, a, b)
}

func OR(a, b string) string {
	if strings.Contains(a, "AND") || strings.Contains(a, "OR") {
		a = fmt.Sprintf("(%s)", a)
	}

	if strings.Contains(b, "AND") || strings.Contains(b, "OR") {
		b = fmt.Sprintf("(%s)", b)
	}

	return fmt.Sprintf(`%s OR %s`, a, b)
}

func GT(name WorkflowFilterName, value string) string {
	return fmt.Sprintf("%s > '%s'", name, strings.Trim(value, "'"))
}

func LT(name WorkflowFilterName, value string) string {
	return fmt.Sprintf("%s < '%s'", name, strings.Trim(value, "'"))
}

func EQ(name WorkflowFilterName, value string) string {
	return fmt.Sprintf("%s = '%s'", name, strings.Trim(value, "'"))
}

func StartsWith(name WorkflowFilterName, value string) string {
	return fmt.Sprintf("%s STARTS_WITH '%s'", name, strings.Trim(value, "'"))
}

func IN(name WorkflowFilterName, value ...string) string {
	return fmt.Sprintf("%s IN (%s)", name, strings.Join(
		utils.Map(
			func(in string) string {
				return fmt.Sprintf("'%s'", strings.Trim(in, "'"))
			},
			value,
		), ", "))
}
