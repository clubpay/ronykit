package util

import (
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"sync"
)

/*
   Creation Time: 2019 - Oct - 03
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
*/

var poolSha512 = sync.Pool{
	New: func() any {
		return sha512.New()
	},
}

// Sha512 appends a 64bytes array which is sha512(in) to out and returns out.
func Sha512(in, out []byte) ([]byte, error) {
	h := poolSha512.Get().(hash.Hash) //nolint:forcetypeassert
	if _, err := h.Write(in); err != nil {
		h.Reset()
		poolSha512.Put(h)

		return out, err
	}
	out = h.Sum(out)
	h.Reset()
	poolSha512.Put(h)

	return out, nil
}

// MustSha512 is Sha512 but it panics if any error happens.
func MustSha512(in, out []byte) []byte {
	var err error
	out, err = Sha512(in, out)
	if err != nil {
		panic(err)
	}

	return out
}

var poolSha256 = sync.Pool{
	New: func() any {
		return sha256.New()
	},
}

// Sha256 appends a 32bytes array which is sha256(in) to out.
func Sha256(in, out []byte) ([]byte, error) {
	h := poolSha256.Get().(hash.Hash) //nolint:forcetypeassert
	if _, err := h.Write(in); err != nil {
		h.Reset()
		poolSha256.Put(h)

		return out, err
	}
	out = h.Sum(out)
	h.Reset()
	poolSha256.Put(h)

	return out, nil
}

func MustSha256(in, out []byte) []byte {
	var err error
	out, err = Sha256(in, out)
	if err != nil {
		panic(err)
	}

	return out
}
