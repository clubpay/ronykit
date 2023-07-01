package meshcluster_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMeshCluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MeshCluster Suite")
}
