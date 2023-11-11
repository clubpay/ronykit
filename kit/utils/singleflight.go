package utils

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
)

// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// errGoexit indicates the runtime.Goexit was called in
// the user given function.
var errGoexit = errors.New("runtime.Goexit was called")

// A panicError is an arbitrary value recovered from a panic
// with the stack trace during the execution of given function.
type panicError struct {
	value any
	stack []byte
}

// Error implements error interface.
func (p *panicError) Error() string {
	return fmt.Sprintf("%v\n\n%s", p.value, p.stack)
}

func (p *panicError) Unwrap() error {
	err, ok := p.value.(error)
	if !ok {
		return nil
	}

	return err
}

func newPanicError(v any) error {
	stack := debug.Stack()

	// The first line of the stack trace is of the form "goroutine N [status]:"
	// but by the time the panic reaches Do the goroutine may no longer exist
	// and its status will have changed. Trim out the misleading line.
	if line := bytes.IndexByte(stack[:], '\n'); line >= 0 {
		stack = stack[line+1:]
	}

	return &panicError{value: v, stack: stack}
}

// call is an in-flight or completed singleflight.Do call
type call struct {
	wg sync.WaitGroup

	// These fields are written once before the WaitGroup is done
	// and are only read after the WaitGroup is done.
	val any
	err error

	// These fields are read and written with the singleflight
	// mutex held before the WaitGroup is done, and are read but
	// not written after the WaitGroup is done.
	dups  int
	chans []chan<- Result
}

// Result holds the results of Do, so they can be passed
// on a channel.
type Result struct {
	Val    any
	Err    error
	Shared bool
}

type SingleFlightCall[T any] func(fn func() (T, error)) (T, error)

// SingleFlight executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a
// time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
// The return value shared indicates whether v was given to multiple cal
func SingleFlight[T any](_ func() (T, error)) SingleFlightCall[T] {
	mu := sync.Mutex{}
	var (
		c     *call
		ready = true
	)

	doCall := genDoCall[T](&mu, &ready)

	return func(fn func() (T, error)) (T, error) {
		mu.Lock()
		if ready {
			ready = false
			c = new(call)
			c.wg.Add(1)
			mu.Unlock()

			doCall(c, fn)

			return c.val.(T), c.err //nolint:forcetypeassert
		}

		c.dups++
		mu.Unlock()
		c.wg.Wait()

		var e *panicError
		if c.err != nil && errors.As(c.err, &e) {
			panic(e)
		}

		if errors.Is(c.err, errGoexit) {
			runtime.Goexit()
		}

		return c.val.(T), c.err //nolint:forcetypeassert
	}
}

// doCall handles the single call for a key.
//
//nolint:gocognit
func genDoCall[T any](mu *sync.Mutex, ready *bool) func(c *call, fn func() (T, error)) {
	return func(c *call, fn func() (T, error)) {
		normalReturn := false
		recovered := false

		// use double-defer to distinguish panic from runtime.Goexit,
		// more details see https://golang.org/cl/134395
		defer func() {
			// the given function invoked runtime.Goexit
			if !normalReturn && !recovered {
				c.err = errGoexit
			}

			mu.Lock()
			defer mu.Unlock()
			c.wg.Done()
			*ready = true

			if e, ok := c.err.(*panicError); ok {
				// To prevent the waiting channels from being blocked forever,
				// needs to ensure that this panic cannot be recovered.
				if len(c.chans) > 0 {
					go panic(e)
					select {} // Keep this goroutine around so that it will appear in the crash dump.
				} else {
					panic(e)
				}
			} else if errors.Is(c.err, errGoexit) {
				// Already in the process of goexit, no need to call again
			} else {
				// Normal return
				for _, ch := range c.chans {
					ch <- Result{c.val, c.err, c.dups > 0}
				}
			}
		}()

		func() {
			defer func() {
				if !normalReturn {
					// Ideally, we would wait to take a stack trace until we've determined
					// whether this is a panic or a runtime.Goexit.
					//
					// Unfortunately, the only way we can distinguish the two is to see
					// whether the recover stopped the goroutine from terminating, and by
					// the time we know that, the part of the stack trace relevant to the
					// panic has been discarded.
					if r := recover(); r != nil {
						c.err = newPanicError(r)
					}
				}
			}()

			c.val, c.err = fn()
			normalReturn = true
		}()

		if !normalReturn {
			recovered = true
		}
	}
}
