package flow

import (
	"context"
	"testing"

	"go.temporal.io/sdk/testsuite"
)

func TestGetHeartbeatTyped(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.SetHeartbeatDetails(42)

	fn := func(ctx context.Context) (int, error) {
		ac := &ActivityContext[struct{}, int, EMPTY]{ctx: ctx}

		return GetHeartbeat[int, struct{}, int, EMPTY](ac)
	}
	env.RegisterActivity(fn)

	val, err := env.ExecuteActivity(fn)
	if err != nil {
		t.Fatal(err)
	}

	var got int
	if err := val.Get(&got); err != nil {
		t.Fatal(err)
	}
	if got != 42 {
		t.Fatalf("got %d", got)
	}
}

func TestHasHeartbeat(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.SetHeartbeatDetails("resume")

	fn := func(ctx context.Context) (bool, error) {
		ac := &ActivityContext[struct{}, bool, EMPTY]{ctx: ctx}
		if !ac.HasHeartbeat() {
			t.Fatal("expected heartbeat details")
		}

		var details string
		if err := ac.HeartbeatDetails(&details); err != nil {
			return false, err
		}
		if details != "resume" {
			t.Fatalf("details %q", details)
		}

		return true, nil
	}
	env.RegisterActivity(fn)

	val, err := env.ExecuteActivity(fn)
	if err != nil {
		t.Fatal(err)
	}

	var ok bool
	if err := val.Get(&ok); err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected true")
	}
}
