package httpmux_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHttpmux(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Httpmux Suite")
}
