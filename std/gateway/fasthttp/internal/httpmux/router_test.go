package httpmux_test

import (
	"github.com/clubpay/ronykit/std/gateway/fasthttp/internal/httpmux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/valyala/fasthttp"
)

var _ = Describe("Router", func() {
	mux := &httpmux.Mux{}
	rd := &httpmux.RouteData{}
	mux.POST("/r1/:p1/something", rd)
	mux.GET("/r1/:p1/something", rd)

	It("Wildcard route must match with GET", func() {
		rd, p, _ := mux.Lookup(fasthttp.MethodGet, "/r1/x/something")
		Expect(p.ByName("p1")).To(Equal("x"))
		Expect(rd).To(BeEquivalentTo(rd))
	})

	It("Wildcard route must match with POST", func() {
		rd, p, _ := mux.Lookup(fasthttp.MethodPost, "/r1/x/something")
		Expect(p.ByName("p1")).To(Equal("x"))
		Expect(rd).To(BeEquivalentTo(rd))
	})
})
