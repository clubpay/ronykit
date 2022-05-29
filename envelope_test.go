package ronykit_test

import (
	"github.com/clubpay/ronykit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Envelope", func() {
	tc := NewTestConn(1, "", false)
	ctx := ronykit.NewContext()
	e := ronykit.NewEnvelope(ctx, tc, true)
	e.
		SetHdr("K1", "V1").
		SetHdr("K2", "V2").
		SetMsg(ronykit.RawMessage("Some Random Message"))

	It("should read header and message", func() {
		Expect(e.GetHdr("K1")).To(Equal("V1"))
		Expect(e.GetHdr("K2")).To(Equal("V2"))
		Expect(e.GetMsg()).To(BeEquivalentTo("Some Random Message"))
	})
})
