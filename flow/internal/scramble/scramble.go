package scramble

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash/crc32"
)

var ErrCiphertextTooShort = errors.New("scramble: ciphertext too short")

type Scramble struct {
	secretKey []byte
	aead      cipher.AEAD
	nonceSize int
	tagSize   int
	ad        [4]byte
}

func NewScramble(seed string) (*Scramble, error) {
	s := &Scramble{secretKey: aesKey(seed)}

	b, err := aes.NewCipher(s.secretKey)
	if err != nil {
		return nil, err
	}

	s.aead, err = cipher.NewGCM(b)
	if err != nil {
		return nil, err
	}

	s.nonceSize = s.aead.NonceSize()
	s.tagSize = s.aead.Overhead()
	binary.BigEndian.PutUint32(s.ad[:], s.Hash())

	return s, nil
}

func MustNewScramble(seed string) *Scramble {
	s, err := NewScramble(seed)
	if err != nil {
		panic(err)
	}

	return s
}

func aesKey(seed string) []byte {
	key := []byte(seed)
	switch len(key) {
	case 16, 24, 32:
		return key
	default:
		sum := sha256.Sum256(key)

		return sum[:]
	}
}

func (s *Scramble) Hash() uint32 {
	csSha256 := sha256.Sum256(s.secretKey)

	return crc32.ChecksumIEEE(csSha256[:])
}

// Encrypt encrypts src. If you want to prevent memory allocation,
// dst must have capacity of len(src) + NonceSize + Overhead.
func (s *Scramble) Encrypt(src, dst []byte) ([]byte, error) {
	c := s.nonceSize + s.tagSize + len(src)
	if cap(dst) < c {
		dst = make([]byte, s.nonceSize, c)
	} else {
		dst = dst[:s.nonceSize]
	}

	if _, err := rand.Read(dst[:s.nonceSize]); err != nil {
		return nil, err
	}

	return s.aead.Seal(dst, dst[:s.nonceSize], src, s.ad[:]), nil
}

// Decrypt decrypts src. If you want to prevent memory allocation,
// dst must have capacity of len(src) - NonceSize.
func (s *Scramble) Decrypt(src, dst []byte) ([]byte, error) {
	if len(src) < s.nonceSize+s.tagSize {
		return nil, ErrCiphertextTooShort
	}

	c := len(src) - s.nonceSize
	if cap(dst) < c {
		dst = make([]byte, 0, c)
	} else {
		dst = dst[:0]
	}

	return s.aead.Open(dst, src[:s.nonceSize], src[s.nonceSize:], s.ad[:])
}

func (s *Scramble) Overhead() int {
	return s.tagSize
}

func (s *Scramble) NonceSize() int {
	return s.nonceSize
}
