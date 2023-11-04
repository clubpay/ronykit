package rony

import (
	"reflect"
)

type InitState[S State[A], A Action] func() S

func ToInitiateState[S State[A], A Action](s S) InitState[S, A] {
	return func() S {
		return s
	}
}

type SetupContext[S State[A], A Action] struct {
	s   *S
	cfg *serverConfig
}

// Setup is a helper function to set up server and services.
// Make sure S is a pointer to a struct, otherwise this function panics
func Setup[S State[A], A Action](srv *Server, stateFactory InitState[S, A]) *SetupContext[S, A] {
	state := stateFactory()
	if reflect.TypeOf(state).Kind() != reflect.Ptr {
		panic("state must be a pointer to a struct")
	}
	if reflect.TypeOf(state).Elem().Kind() != reflect.Struct {
		panic("state must be a pointer to a struct")
	}

	ctx := &SetupContext[S, A]{
		s:   &state,
		cfg: &srv.cfg,
	}

	return ctx
}
