package flow

import (
	"reflect"
	"testing"
)

type factoryNoSelfRegState struct{}
type factoryIdempotentState struct{}

func TestActivityFactoryDoesNotSelfRegister(t *testing.T) {
	stateT := reflect.TypeOf(factoryNoSelfRegState{})

	registryMu.Lock()
	before := len(registeredActivities[stateT])
	registryMu.Unlock()

	_ = NewActivityFactory[int, int, factoryNoSelfRegState](
		"FactoryActNoSelfReg", "factory-group",
		func(factoryNoSelfRegState) ActivityFunc[int, int, factoryNoSelfRegState] {
			return func(_ *ActivityContext[int, int, factoryNoSelfRegState], req int) (*int, error) {
				v := req + 1

				return &v, nil
			}
		},
	)

	registryMu.Lock()
	after := len(registeredActivities[stateT])
	registryMu.Unlock()

	if after != before {
		t.Fatalf("factory registered %d activities directly, want 0 new", after-before)
	}
}

func TestActivityFactoryInitIsIdempotent(t *testing.T) {
	_ = NewActivityFactory[int, int, factoryIdempotentState](
		"FactoryActIdempotent", "factory-group",
		func(factoryIdempotentState) ActivityFunc[int, int, factoryIdempotentState] {
			return func(_ *ActivityContext[int, int, factoryIdempotentState], req int) (*int, error) {
				return &req, nil
			}
		},
	)

	b := &mockBackend{group: "factory-group", taskQ: "q"}
	sdk := NewSDK(SDKConfig{DefaultBackend: b})
	sdk.InitWithState(factoryIdempotentState{})

	if len(b.actRegs) != 1 {
		t.Fatalf("first init registered %d activities, want 1: %v", len(b.actRegs), b.actRegs)
	}

	sdk.InitWithState(factoryIdempotentState{})
	if len(b.actRegs) != 1 {
		t.Fatalf("second init registered %d activities, want 1: %v", len(b.actRegs), b.actRegs)
	}
}
