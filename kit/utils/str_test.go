package utils_test

import (
	"testing"
	"unsafe"

	"github.com/clubpay/ronykit/kit/utils"
)

func TestClone(t *testing.T) {
	x := utils.RandomID(128)
	y := utils.CloneStr(x)
	if unsafe.Pointer(&x) == unsafe.Pointer(&y) {
		t.Fatal("CloneStr should return a new string", unsafe.Pointer(&x), unsafe.Pointer(&y))
	}
	tt := utils.CloneBytes(utils.S2B(y))
	if unsafe.Pointer(&tt) == unsafe.Pointer(&y) {
		t.Fatal("CloneBytes should return a new slice", unsafe.Pointer(&tt), unsafe.Pointer(&y))
	}
}

func BenchmarkConvert(b *testing.B) {
	x := utils.RandomID(128)
	for i := 0; i < b.N; i++ {
		y := utils.B2S(utils.S2B(x))
		if len(x) != len(y) {
			b.Fatal("B2S should return same length")
		}
	}
}
