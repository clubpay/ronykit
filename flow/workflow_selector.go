package flow

import "go.temporal.io/sdk/workflow"

type Selector workflow.Selector

// SelectorAddReceive registers a callback function to be called when a channel has a message to receive.
// The callback is called when Select(ctx) is called.
// The message is expected be consumed by the callback function.
// The branch is automatically removed after the channel is closed and callback function is called once
// with more parameter set to false.
func SelectorAddReceive[T any](s Selector, ch Channel[T], fn func(ch Channel[T], more bool)) {
	s.AddReceive(ch.ch, func(c workflow.ReceiveChannel, more bool) {
		fn(ch, more)
	})
}

// SelectorAddFuture registers a callback function to be called when a future is ready.
// The callback is called when Select(ctx) is called.
// The callback is called once per ready future even if Select is called multiple times for the same
// Selector instance.
func SelectorAddFuture[T any](s Selector, f Future[T], fn func(f Future[T])) {
	s.AddFuture(f.f, func(wf workflow.Future) {
		fn(f)
	})
}
