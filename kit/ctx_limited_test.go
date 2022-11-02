package kit_test

import (
	"github.com/clubpay/ronykit/kit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CtxLimited", func() {
	ctx := kit.NewContext()
	conn := newTestConn(100, "", true)
	ctx.SetConn(conn)

	It("Should be synced with the main context", func() {
		limitCtx := ctx.Limited()
		Expect(limitCtx.Conn().ConnID()).To(Equal(conn.ConnID()))
		Expect(limitCtx.ServiceName()).To(Equal(ctx.ServiceName()))
		limitCtx.In().SetHdr("k1", "v1")
		Expect(ctx.In().GetHdr("k1")).To(Equal("v1"))
	})
})
