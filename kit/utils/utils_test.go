package utils_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGateway(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "RonyKit/Kit/Utils Suite")
}
