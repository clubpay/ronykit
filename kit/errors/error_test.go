package errors_test

import (
	builtinErr "errors"
	"fmt"
	"testing"

	"github.com/clubpay/ronykit/kit/errors"
	"github.com/stretchr/testify/assert"
)

var (
	errInternal  = builtinErr.New("internal error")
	errRuntime   = builtinErr.New("runtime error")
	errEndOfFile = builtinErr.New("end of file")

	errWrappedInternal = fmt.Errorf("wrapped error: %w", errInternal)
)

func TestWrapError(t *testing.T) {
	we := errors.Wrap(errRuntime, errEndOfFile)

	t.Run("Wrap 1", func(t *testing.T) {
		assert.True(t, builtinErr.Is(we, errRuntime))
		assert.True(t, builtinErr.Is(we, errEndOfFile))
		assert.Equal(t, "runtime error: end of file", we.Error())
	})

	t.Run("Wrap 2", func(t *testing.T) {
		wwe := errors.Wrap(errInternal, we)
		assert.True(t, builtinErr.Is(wwe, errInternal))
		assert.True(t, builtinErr.Is(wwe, errRuntime))
		assert.True(t, builtinErr.Is(wwe, errEndOfFile))
		assert.Equal(t, "internal error: runtime error: end of file", wwe.Error())
	})

	t.Run("Wrap 3", func(t *testing.T) {
		wwi := errors.Wrap(errWrappedInternal, we)
		assert.True(t, builtinErr.Is(wwi, errWrappedInternal))
		assert.True(t, builtinErr.Is(wwi, errInternal))
		assert.True(t, builtinErr.Is(wwi, errRuntime))
		assert.True(t, builtinErr.Is(wwi, errEndOfFile))
		assert.Equal(t, "wrapped error: internal error: runtime error: end of file", wwi.Error())
	})
}
