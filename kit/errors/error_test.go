package errors_test

import (
	builtinErr "errors"
	"fmt"
	"testing"

	"github.com/clubpay/ronykit/kit/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	errInternal  = builtinErr.New("internal error")
	errRuntime   = builtinErr.New("runtime error")
	errEndOfFile = builtinErr.New("end of file")

	errWrappedInternal = fmt.Errorf("wrapped error: %w", errInternal)
)

func TestError(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ronykit Suite")
}

var _ = Describe("WrapError", func() {
	we := errors.Wrap(errRuntime, errEndOfFile)

	It("Wrap 1", func() {
		Expect(builtinErr.Is(we, errRuntime)).To(BeTrue())
		Expect(builtinErr.Is(we, errEndOfFile)).To(BeTrue())
		Expect(we.Error()).To(BeEquivalentTo("runtime error: end of file"))
	})

	It("Wrap 2", func() {
		wwe := errors.Wrap(errInternal, we)
		Expect(builtinErr.Is(wwe, errInternal)).To(BeTrue())
		Expect(builtinErr.Is(wwe, errRuntime)).To(BeTrue())
		Expect(builtinErr.Is(wwe, errEndOfFile)).To(BeTrue())
		Expect(wwe.Error()).To(BeEquivalentTo("internal error: runtime error: end of file"))
	})

	It("Wrap 3", func() {
		wwi := errors.Wrap(errWrappedInternal, we)
		Expect(builtinErr.Is(wwi, errWrappedInternal)).To(BeTrue())
		Expect(builtinErr.Is(wwi, errInternal)).To(BeTrue())
		Expect(builtinErr.Is(wwi, errRuntime)).To(BeTrue())
		Expect(builtinErr.Is(wwi, errEndOfFile)).To(BeTrue())
		Expect(wwi.Error()).To(BeEquivalentTo("wrapped error: internal error: runtime error: end of file"))
	})
})
