package scramble

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"hash/crc32"
)

type Scramble struct {
	secretKey []byte
	aead      cipher.AEAD
	nonceSize int
	tagSize   int
	ad        [4]byte
}

func NewScramble(seed string) *Scramble {
	s := &Scramble{}
	s.secretKey = []byte(seed)
	b, err := aes.NewCipher(s.secretKey)
	if err != nil {
		panic(err)
	}
	s.aead, err = cipher.NewGCM(b)
	if err != nil {
		panic(err)
	}
	s.nonceSize = s.aead.NonceSize()
	s.tagSize = s.aead.Overhead()
	binary.BigEndian.PutUint32(s.ad[:], s.Hash())

	return s
}

func (s *Scramble) Hash() uint32 {
	csSha256 := sha256.Sum256(s.secretKey)

	return crc32.ChecksumIEEE(csSha256[:])
}

// Encrypt encrypts src. If you want to prevent memory allocation,
// dst must have capacity of len(src) + NonceSize + Overhead.
func (s *Scramble) Encrypt(src, dst []byte) []byte {
	c := s.nonceSize + s.tagSize + len(src)
	if cap(dst) < c {
		dst = make([]byte, s.nonceSize, c)
	} else {
		dst = dst[:s.nonceSize]
	}
	copy(dst[:4], s.ad[:])
	_, _ = rand.Read(dst[4:])

	return s.aead.Seal(dst, dst[:s.nonceSize], src, s.ad[:])
}

// Decrypt decrypts src. If you want to prevent memory allocation,
// dst must have capacity of len(src) - NonceSize.
func (s *Scramble) Decrypt(src, dst []byte) ([]byte, error) {
	var (
		err error
		c   = len(src) - s.nonceSize
	)
	if cap(dst) < c {
		dst = make([]byte, 0, c)
	} else {
		dst = dst[:0]
	}
	dst, err = s.aead.Open(dst, src[:s.nonceSize], src[s.nonceSize:], s.ad[:])

	return dst, err
}

func (s *Scramble) Overhead() int {
	return s.tagSize
}

func (s *Scramble) NonceSize() int {
	return s.nonceSize
}
