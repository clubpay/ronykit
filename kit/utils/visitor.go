package utils

// VisitAll runs all visitors and returns the final state
func VisitAll[VisitorState any](
	initial VisitorState,
	visitors ...func(ctx *VisitorState),
) VisitorState {
	state := initial
	for _, visitor := range visitors {
		visitor(&state)
	}

	return state
}

// VisitCond runs all visitors if the condition is true and returns the final state
// If the condition is false, the visitor will stop and DO NOT run the rest of the visitors.
// cond function is called before each visitor.
//
// NOTE: `cond` is called before each visitor, hence, it will be run on initial state too.
func VisitCond[VisitorState any](
	initial VisitorState,
	cond func(ctx *VisitorState) bool,
	visitors ...func(ctx *VisitorState),
) VisitorState {
	state := initial
	for _, visitor := range visitors {
		if cond(&state) {
			visitor(&state)
		}
	}

	return state
}

// VisitStopOnErr runs all visitors and returns the final state
// If any of the visitors returns an error, the visitor will stop and DO NOT run the rest of the visitors.
// It returns the latest state and the error.
func VisitStopOnErr[VisitorState any](
	initial VisitorState,
	visitors ...func(ctx *VisitorState) error,
) (VisitorState, error) {
	state := initial
	for _, visitor := range visitors {
		err := visitor(&state)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}
