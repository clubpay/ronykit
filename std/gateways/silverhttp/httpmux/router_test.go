package httpmux_test

import (
	"github.com/clubpay/ronykit/std/gateways/silverhttp"
	"github.com/clubpay/ronykit/std/gateways/silverhttp/httpmux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Router", func() {
	mux := &httpmux.Mux{}
	rd := &httpmux.RouteData{}
	mux.POST("/r1/:p1/something", rd)
	mux.GET("/r1/:p1/something", rd)

	It("Wildcard route must match with GET", func() {
		rd, p, _ := mux.Lookup(silverhttp.MethodGet, "/r1/x/something")
		Expect(p.ByName("p1")).To(Equal("x"))
		Expect(rd).To(BeEquivalentTo(rd))
	})

	It("Wildcard route must match with POST", func() {
		rd, p, _ := mux.Lookup(silverhttp.MethodPost, "/r1/x/something")
		Expect(p.ByName("p1")).To(Equal("x"))
		Expect(rd).To(BeEquivalentTo(rd))
	})
})
