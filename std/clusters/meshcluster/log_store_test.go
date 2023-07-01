package meshcluster_test

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/std/clusters/meshcluster"
	"github.com/hashicorp/raft"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LogStore", func() {
	Describe("Basic Functionality", basicFunctionality)
	Describe("Concurrent Logs", concurrentLogs)
})

var basicFunctionality = func() {
	dbPath := fmt.Sprintf("_testdb_%s", utils.RandomID(5))
	AfterEach(func() {
		DeferCleanup(func() {
			_ = os.RemoveAll(dbPath)
		})
	})

	logStore, err := meshcluster.NewLogStore(dbPath)
	It("should return badgerLogStore", func() {
		Expect(err).To(BeNil())
		Expect(logStore).ShouldNot(BeNil())
	})

	It("Should update First and Last Index", func() {
		Expect(logStore.StoreLog(randomLog(3))).To(BeNil())
		Expect(logStore.FirstIndex()).To(Equal(uint64(3)))
		Expect(logStore.LastIndex()).To(Equal(uint64(3)))
		Expect(logStore.StoreLog(randomLog(5))).To(BeNil())
		Expect(logStore.FirstIndex()).To(Equal(uint64(3)))
		Expect(logStore.LastIndex()).To(Equal(uint64(5)))
		Expect(logStore.StoreLog(randomLog(4))).To(BeNil())
		Expect(logStore.FirstIndex()).To(Equal(uint64(3)))
		Expect(logStore.LastIndex()).To(Equal(uint64(5)))
		Expect(logStore.StoreLog(randomLog(10))).To(BeNil())
		Expect(logStore.FirstIndex()).To(Equal(uint64(3)))
		Expect(logStore.LastIndex()).To(Equal(uint64(10)))
		Expect(logStore.StoreLog(randomLog(1))).To(BeNil())
		Expect(logStore.FirstIndex()).To(Equal(uint64(1)))
		Expect(logStore.LastIndex()).To(Equal(uint64(10)))

		logs := []*raft.Log{
			randomLog(2), randomLog(3), randomLog(4), randomLog(5), randomLog(6),
			randomLog(13), randomLog(14), randomLog(15), randomLog(16), randomLog(17),
		}
		Expect(logStore.StoreLogs(logs)).To(BeNil())
		Expect(logStore.FirstIndex()).To(Equal(uint64(1)))
		Expect(logStore.LastIndex()).To(Equal(uint64(17)))
	})
}

var concurrentLogs = func() {
	dbPath := fmt.Sprintf("_testdb_%s", utils.RandomID(5))
	AfterEach(func() {
		DeferCleanup(func() {
			_ = os.RemoveAll(dbPath)
		})
	})

	logStore, err := meshcluster.NewLogStore(dbPath)

	It("should return badgerLogStore", func() {
		Expect(err).To(BeNil())
		Expect(logStore).ShouldNot(BeNil())
	})

	It("Should update First and Last Index", func() {
		wg := sync.WaitGroup{}
		for i := 1; i <= 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer GinkgoRecover()
				defer wg.Done()
				Expect(logStore.StoreLog(randomLog(uint64(idx)))).To(BeNil())
			}(i)
		}
		wg.Wait()
		Expect(logStore.FirstIndex()).To(Equal(uint64(1)))
		Expect(logStore.LastIndex()).To(Equal(uint64(20)))
	})
}

func randomLog(idx uint64) *raft.Log {
	return &raft.Log{
		Index:      idx,
		Term:       0,
		Type:       0,
		Data:       utils.S2B(utils.RandomID(128)),
		Extensions: nil,
		AppendedAt: time.Time{},
	}
}
