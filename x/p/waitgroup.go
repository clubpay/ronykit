package p

import (
	"sync"
)

/*
   Creation Time: 2019 - Oct - 03
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
*/

var waitGroupPool sync.Pool

func AcquireWaitGroup() *sync.WaitGroup {
	wgv := waitGroupPool.Get()
	if wgv == nil {
		return &sync.WaitGroup{}
	}

	return wgv.(*sync.WaitGroup) //nolint:forcetypeassert
}

func ReleaseWaitGroup(wg *sync.WaitGroup) {
	waitGroupPool.Put(wg)
}
