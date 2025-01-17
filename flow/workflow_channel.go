package flow

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

func NewChannel[REQ, RES, T any](ctx *WorkflowContext[REQ, RES], fn func() T) (T, error) {
	reqEncoded := workflow.SideEffect(
		ctx.Context(),
		func(wctx workflow.Context) any {
			return fn()
		},
	)

	var out T
	err := reqEncoded.Get(&out)

	return out, err
}

type Channel[T any] struct {
	ch workflow.Channel
}

// Send blocks until the data is sent.
func (ch Channel[T]) Send(ctx Context, v T) {
	ch.ch.Send(ctx, v)
}

// SendAsync try to send without blocking. It returns true if the data was sent, otherwise it returns false.
func (ch Channel[T]) SendAsync(v T) (ok bool) {
	return ch.ch.SendAsync(v)
}

// Close the Channel, and prohibit subsequent sends.
func (ch Channel[T]) Close() {
	ch.ch.Close()
}

// Name returns the name of the Channel.
// If the Channel was retrieved from a GetSignalChannel call, Name returns the signal name.
//
// A Channel created without an explicit name will use a generated name by the SDK and
// is not deterministic.
func (ch Channel[T]) Name() string { return ch.ch.Name() }

// Receive blocks until it receives a value, and then assigns the received value to the provided pointer.
// Returns false when Channel is closed.
// Parameter valuePtr is a pointer to the expected data structure to be received. For example:
//
//	var v string
//	c.Receive(ctx, &v)
//
// Note, values should not be reused for extraction here because merging on
// top of existing values may result in unexpected behavior similar to
// json.Unmarshal.
func (ch Channel[T]) Receive(ctx Context, valuePtr *T) (more bool) {
	return ch.ch.Receive(ctx, valuePtr)
}

// ReceiveWithTimeout blocks up to timeout until it receives a value, and then assigns the received value to the
// provided pointer.
// Returns more value of false when Channel is closed.
// Returns ok value of false when no value was found in the channel for the duration of timeout or
// the ctx was canceled.
// The valuePtr is not modified if ok is false.
// Parameter valuePtr is a pointer to the expected data structure to be received. For example:
//
//	var v string
//	c.ReceiveWithTimeout(ctx, time.Minute, &v)
//
// Note, values should not be reused for extraction here because merging on
// top of existing values may result in unexpected behavior similar to
// json.Unmarshal.
func (ch Channel[T]) ReceiveWithTimeout(ctx Context, timeout time.Duration, valuePtr *T) (ok, more bool) {
	return ch.ch.ReceiveWithTimeout(ctx, timeout, valuePtr)
}

// ReceiveAsync try to receive from Channel without blocking. If there is data available from the Channel, it
// assign the data to valuePtr and returns true. Otherwise, it returns false immediately.
//
// Note, values should not be reused for extraction here because merging on
// top of existing values may result in unexpected behavior similar to
// json.Unmarshal.
func (ch Channel[T]) ReceiveAsync(valuePtr *T) (ok bool) {
	return ch.ch.ReceiveAsync(valuePtr)
}

// ReceiveAsyncWithMoreFlag is same as ReceiveAsync with extra return value more to indicate if there could be
// more value from the Channel. The more is false when Channel is closed.
//
// Note, values should not be reused for extraction here because merging on
// top of existing values may result in unexpected behavior similar to
// json.Unmarshal.
func (ch Channel[T]) ReceiveAsyncWithMoreFlag(valuePtr *T) (ok bool, more bool) {
	return ch.ch.ReceiveAsyncWithMoreFlag(valuePtr)
}

// Len returns the number of buffered messages plus the number of blocked Send calls.
func (ch Channel[T]) Len() int {
	return ch.ch.Len()
}
