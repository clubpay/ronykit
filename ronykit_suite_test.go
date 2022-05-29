package ronykit_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRonykit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ronykit Suite")
}
