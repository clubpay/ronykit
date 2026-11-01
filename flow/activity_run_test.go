package flow

import (
	"testing"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
)

func TestToClientStartActivityOptions(t *testing.T) {
	opts := toClientStartActivityOptions(StartActivityOptions{
		ID:                       "act-1",
		StartToCloseTimeout:      5 * time.Second,
		ActivityIDReusePolicy:    ActivityIDReusePolicyRejectDuplicate,
		ActivityIDConflictPolicy: ActivityIDConflictPolicyUseExisting,
		Summary:                  "charge",
		StaticDetails:            "order 7",
		StartDelay:               time.Second,
	}, "queue")

	if opts.ID != "act-1" {
		t.Fatalf("id %q", opts.ID)
	}
	if opts.TaskQueue != "queue" {
		t.Fatalf("task queue %q", opts.TaskQueue)
	}
	if opts.StartToCloseTimeout != 5*time.Second {
		t.Fatalf("start to close %s", opts.StartToCloseTimeout)
	}
	if opts.ScheduleToCloseTimeout != 24*time.Hour {
		t.Fatalf("schedule to close %s", opts.ScheduleToCloseTimeout)
	}
	if opts.ActivityIDReusePolicy != enumspb.ACTIVITY_ID_REUSE_POLICY_REJECT_DUPLICATE {
		t.Fatalf("reuse policy %v", opts.ActivityIDReusePolicy)
	}
	if opts.ActivityIDConflictPolicy != enumspb.ACTIVITY_ID_CONFLICT_POLICY_USE_EXISTING {
		t.Fatalf("conflict policy %v", opts.ActivityIDConflictPolicy)
	}
	if opts.Summary != "charge" || opts.StaticDetails != "order 7" || opts.StartDelay != time.Second {
		t.Fatalf("options %+v", opts)
	}
}

func TestToClientStartActivityOptionsDefaults(t *testing.T) {
	first := toClientStartActivityOptions(StartActivityOptions{TaskQueue: "other"}, "queue")
	second := toClientStartActivityOptions(StartActivityOptions{}, "queue")

	if first.TaskQueue != "other" {
		t.Fatalf("task queue %q", first.TaskQueue)
	}
	if second.TaskQueue != "queue" {
		t.Fatalf("default task queue %q", second.TaskQueue)
	}
	if first.ID == "" || second.ID == "" || first.ID == second.ID {
		t.Fatalf("expected generated ids, got %q and %q", first.ID, second.ID)
	}
}
