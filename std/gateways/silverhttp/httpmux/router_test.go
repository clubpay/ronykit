package httpmux_test

import (
	"testing"

	"github.com/clubpay/ronykit/std/gateways/silverhttp"
	"github.com/clubpay/ronykit/std/gateways/silverhttp/httpmux"
	"github.com/stretchr/testify/assert"
)

func TestRouter(t *testing.T) {
	mux := &httpmux.Mux{}
	expectedRD := &httpmux.RouteData{}
	mux.POST("/r1/:p1/something", expectedRD)
	mux.GET("/r1/:p1/something", expectedRD)

	t.Run("Wildcard route must match with GET", func(t *testing.T) {
		rd, p, _ := mux.Lookup(silverhttp.MethodGet, "/r1/x/something")
		assert.Equal(t, "x", p.ByName("p1"))
		assert.Equal(t, expectedRD, rd)
	})

	t.Run("Wildcard route must match with POST", func(t *testing.T) {
		rd, p, _ := mux.Lookup(silverhttp.MethodPost, "/r1/x/something")
		assert.Equal(t, "x", p.ByName("p1"))
		assert.Equal(t, expectedRD, rd)
	})
}
