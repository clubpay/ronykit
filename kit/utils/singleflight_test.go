package utils_test

import (
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/kit/utils"
)

func TestSingleFlight(t *testing.T) {
	cnt := 0
	fn := func() (string, error) {
		cnt++
		time.Sleep(time.Millisecond * 500)

		return "hello", nil
	}

	sf := utils.SingleFlight[string]()

	wg := sync.WaitGroup{}
	for j := 0; j < 10; j++ {
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				res, err := sf(fn)
				if err != nil {
					t.Error(err)
				}
				t.Log(res)
			}()
		}
		wg.Wait()

		if cnt != 1 {
			t.Errorf("expected 10, got %d", cnt)
		}

		cnt = 0
	}
}
