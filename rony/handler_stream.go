package rony

type StreamHandler[
	S State[A], A Action,
	IN Message,
] func(ctx *Context[S, A], in IN) error
