package utils

import (
	"crypto/rand"
	"encoding/binary"
	mathRand "math/rand"
	"sync"
	_ "unsafe"
)

/*
   Creation Time: 2019 - Oct - 03
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
*/

func init() {
	rndGen.New = func() any {
		x := mathRand.New(mathRand.NewSource(CPUTicks()))

		return x
	}
}

const (
	digits              = "0123456789"
	digitsLength        = uint32(len(digits))
	alphaNumerics       = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	alphaNumericsLength = uint32(len(alphaNumerics))
)

// FastRand is a fast thread local random function.
//
//go:linkname FastRand runtime.fastrand
func FastRand() uint32

type randomGenerator struct {
	sync.Pool
}

func (rg *randomGenerator) GetRand() *mathRand.Rand {
	return rg.Get().(*mathRand.Rand) //nolint:forcetypeassert
}

func (rg *randomGenerator) PutRand(r *mathRand.Rand) {
	rg.Put(r)
}

var rndGen randomGenerator

// RandomID generates a pseudo-random string with length 'n' which characters are alphanumerics.
func RandomID(n int) string {
	rnd := rndGen.GetRand()
	b := make([]byte, n)

	for i := range b {
		b[i] = alphaNumerics[FastRand()%alphaNumericsLength]
	}

	rndGen.PutRand(rnd)

	return ByteToStr(b)
}

func RandomIDs(n ...int) []string {
	str := make([]string, 0, len(n))
	for _, x := range n {
		str = append(str, RandomID(x))
	}

	return str
}

// RandomDigit generates a pseudo-random string with length 'n' which characters are only digits (0-9)
func RandomDigit(n int) string {
	rnd := rndGen.GetRand()

	b := make([]byte, n)
	for i := 0; i < len(b); i++ {
		b[i] = digits[FastRand()%digitsLength]
	}

	rndGen.PutRand(rnd)

	return ByteToStr(b)
}

// RandomInt64 produces a pseudo-random 63bit number, if n == 0 there will be no limit otherwise
// the output will be smaller than n
func RandomInt64(n int64) (x int64) {
	rnd := rndGen.GetRand()
	if n == 0 {
		x = rnd.Int63()
	} else {
		x = rnd.Int63n(n)
	}

	rndGen.PutRand(rnd)

	return
}

// RandomInt32 produces a pseudo-random 31bit number, if n == 0 there will be no limit otherwise
// the output will be smaller than n
func RandomInt32(n int32) (x int32) {
	rnd := rndGen.GetRand()
	if n == 0 {
		x = rnd.Int31()
	} else {
		x = rnd.Int31n(n)
	}

	rndGen.PutRand(rnd)

	return
}

// SecureRandomInt63 produces a secure pseudo-random 63bit number
func SecureRandomInt63(n int64) (x int64) {
	var b [8]byte

	_, _ = rand.Read(b[:])

	xx := binary.BigEndian.Uint64(b[:])
	if n > 0 {
		x = int64(xx) % n
	} else {
		x = int64(xx >> 1)
	}

	return
}

func RandomInt(n int) (x int) {
	rnd := rndGen.GetRand()
	if n == 0 {
		x = rnd.Int()
	} else {
		x = rnd.Intn(n)
	}

	rndGen.PutRand(rnd)

	return
}

// RandomUint64 produces a pseudo-random unsigned number
func RandomUint64(n uint64) (x uint64) {
	rnd := rndGen.GetRand()
	if n == 0 {
		x = rnd.Uint64()
	} else {
		x = rnd.Uint64() % n
	}

	rndGen.PutRand(rnd)

	return
}

// SecureRandomUint64 produces a secure pseudo-random 64bit number
func SecureRandomUint64() (x uint64) {
	var b [8]byte

	_, _ = rand.Read(b[:])
	x = binary.BigEndian.Uint64(b[:])

	return
}
