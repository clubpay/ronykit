package rony

import (
	"reflect"
)

// State related types
type (
	Action          comparable
	State[A Action] interface {
		Name() string
		Reduce(action A) error
	}

	EMPTY struct{}
	NOP   struct{}
)

func (e EMPTY) Name() string {
	return "EMPTY"
}

func (EMPTY) Reduce(action NOP) error {
	return nil
}

type InitState[S State[A], A Action] func() S

func ToInitiateState[S State[A], A Action](s S) InitState[S, A] {
	return func() S {
		return s
	}
}

// EmptyState is a helper function to create an empty state.
// This is a noop state that does nothing; whenever you don't need a state,
// you can use this function to create one.
func EmptyState() InitState[EMPTY, NOP] {
	return func() EMPTY {
		return EMPTY{}
	}
}

// SetupContext is a context object holds data until the Server
// starts.
// It is used internally to hold state and server configuration.
type SetupContext[S State[A], A Action] struct {
	s    *S
	name string
	cfg  *serverConfig
}

type SetupOption[S State[A], A Action] func(ctx *SetupContext[S, A])

// Setup is a helper function to set up server and services.
// S **MUST** implement State[A] and also **MUST** be a pointer to a struct, otherwise this function panics
// Possible options are:
// - WithState: to set up state
// - WithUnary: to set up unary handler
// - WithStream: to set up stream handler
func Setup[S State[A], A Action](
	srv *Server,
	name string,
	initState InitState[S, A],
	opt ...SetupOption[S, A],
) {
	state := initState()
	if reflect.TypeOf(state) != reflect.TypeOf(EMPTY{}) {
		if reflect.TypeOf(state).Kind() != reflect.Ptr {
			panic("state must be a pointer to a struct")
		}
		if reflect.TypeOf(state).Elem().Kind() != reflect.Struct {
			panic("state must be a pointer to a struct")
		}
	}

	ctx := &SetupContext[S, A]{
		s:    &state,
		name: name,
		cfg:  &srv.cfg,
	}

	for _, o := range opt {
		o(ctx)
	}

	return
}

// WithUnary is a SetupOption to set up unary handler.
// Possible options are:
// - REST: to set up REST handler
// - GET: to set up GET handler
// - POST: to set up POST handler
// - PUT: to set up PUT handler
// - DELETE: to set up DELETE handler
// - PATCH: to set up PATCH handler
// - HEAD: to set up HEAD handler
// - OPTIONS: to set up OPTIONS handler
func WithUnary[S State[A], A Action, IN, OUT Message](
	h UnaryHandler[S, A, IN, OUT],
	opt ...UnaryOption,
) SetupOption[S, A] {
	return func(ctx *SetupContext[S, A]) {
		registerUnary[IN, OUT, S, A](ctx, h, opt...)
	}
}

// WithStream is a SetupOption to set up stream handler.
// Possible options are:
// - RPC: to set up RPC handler
func WithStream[S State[A], A Action, IN, OUT Message](
	h StreamHandler[S, A, IN, OUT],
	opt ...StreamOption,
) SetupOption[S, A] {
	return func(ctx *SetupContext[S, A]) {
		registerStream[IN, OUT, S, A](ctx, h, opt...)
	}
}

func WithMiddleware[S State[A], A Action](
	h ...StatelessMiddleware,
) SetupOption[S, A] {
	return func(ctx *SetupContext[S, A]) {
		registerStatelessMiddleware[S, A](ctx, h...)
	}
}
