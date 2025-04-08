package buf

import (
	"io"
	"sync"
)

const (
	bitSize       = 32 << (^uint(0) >> 63)
	maxIntHeadBit = 1 << (bitSize - 2)
)

type Bytes struct {
	p  *BytesPool
	ri int
	b  []byte
}

func (bb *Bytes) Write(p []byte) (n int, err error) {
	bb.b = append(bb.b, p...)

	return len(p), nil
}

func (bb *Bytes) Read(p []byte) (n int, err error) {
	if bb.ri >= len(bb.b)-1 {
		return 0, io.EOF
	}
	n = copy(p, bb.b[bb.ri:])
	bb.ri += n

	return n, nil
}

func newBytes(p *BytesPool, n, c int) *Bytes {
	if n > c {
		panic("requested length is greater than capacity")
	}

	return &Bytes{p: p, b: make([]byte, n, c)}
}

func (bb *Bytes) Reset() {
	bb.ri = 0
	bb.b = bb.b[:bb.ri]
}

func (bb *Bytes) Bytes() *[]byte {
	return &bb.b
}

func (bb *Bytes) SetBytes(b *[]byte) {
	if b == nil {
		return
	}
	bb.b = *b
}

func (bb *Bytes) Fill(data []byte, start, end int) {
	copy(bb.b[start:end], data)
}

func (bb *Bytes) CopyFromWithOffset(data []byte, offset int) {
	copy(bb.b[offset:], data)
}

func (bb *Bytes) CopyFrom(src []byte) {
	copy(bb.b, src)
}

func (bb *Bytes) CopyTo(dst []byte) []byte {
	copy(dst, bb.b)

	return dst
}

func (bb *Bytes) AppendFrom(src []byte) {
	bb.b = append(bb.b, src...)
}

func (bb *Bytes) AppendTo(dst []byte) []byte {
	dst = append(dst, bb.b...)

	return dst
}

func (bb *Bytes) AppendByte(b byte) {
	bb.b = append(bb.b, b)
}

func (bb *Bytes) AppendString(s string) {
	bb.b = append(bb.b, s...)
}

func (bb *Bytes) Len() int {
	return len(bb.b)
}

func (bb Bytes) Cap() int {
	return cap(bb.b)
}

func (bb *Bytes) Release() {
	bb.Reset()
	bb.p.put(bb)
}

// BytesPool contains the logic of reusing objects distinguishable by size in generic
// way.
type BytesPool struct {
	pool map[int]*sync.Pool
}

var defaultPool = NewBytesPool(32, 1<<20)

// NewBytesPool creates new BytesPool that reuses objects which size is in logarithmic range
// [min, max].
func NewBytesPool(minSize, maxSize int) *BytesPool {
	p := &BytesPool{
		pool: make(map[int]*sync.Pool),
	}
	logarithmicRange(minSize, maxSize, func(n int) {
		p.pool[n] = &sync.Pool{}
	})

	return p
}

// Get returns probably reused slice of bytes with at least capacity of c and
// exactly len of n.
func (p *BytesPool) Get(n, c int) *Bytes {
	if n > c {
		panic("requested length is greater than capacity")
	}

	size := ceilToPowerOfTwo(c)
	if pool := p.pool[size]; pool != nil {
		v := pool.Get()
		if v != nil {
			bb := v.(*Bytes) //nolint:forcetypeassert
			bb.b = bb.b[:n]

			return bb
		} else {
			return newBytes(p, n, size)
		}
	}

	return newBytes(p, n, c)
}

func Get(n, c int) *Bytes {
	return defaultPool.Get(n, c)
}

// GetCap returns probably reused slice of bytes with at least capacity of n.
func (p *BytesPool) GetCap(c int) *Bytes {
	return p.Get(0, c)
}

func GetCap(c int) *Bytes {
	return defaultPool.Get(0, c)
}

// GetLen returns probably reused slice of bytes with at least capacity of n
// and exactly len of n.
func (p *BytesPool) GetLen(n int) *Bytes {
	return p.Get(n, n)
}

func GetLen(n int) *Bytes {
	return defaultPool.Get(n, n)
}

func FromBytes(in []byte) *Bytes {
	b := defaultPool.GetCap(len(in))
	b.AppendFrom(in)

	return b
}

// put returns given Bytes to reuse pool.
// It does not reuse bytes whose size is not power of two or is out of pool
// min/max range.
func (p *BytesPool) put(bb *Bytes) {
	if pool := p.pool[cap(bb.b)]; pool != nil {
		pool.Put(bb)
	}
}

// logarithmicRange iterates from ceil to power of two min to max,
// calling cb on each iteration.
func logarithmicRange(minSize, maxSize int, cb func(int)) {
	if minSize == 0 {
		minSize = 1
	}
	for n := ceilToPowerOfTwo(minSize); n <= maxSize; n <<= 1 {
		cb(n)
	}
}

// ceilToPowerOfTwo returns the least power of two integer values greater than
// or equal to n.
func ceilToPowerOfTwo(n int) int {
	if n&maxIntHeadBit != 0 && n > maxIntHeadBit {
		panic("argument is too large")
	}
	if n <= 2 {
		return n
	}
	n--
	n = fillBits(n)
	n++

	return n
}

func fillBits(n int) int {
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32

	return n
}
