package httpmux_test

import (
	"testing"
)

func TestHttpmux(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Httpmux Suite")
}
