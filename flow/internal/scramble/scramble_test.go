package scramble

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	s := MustNewScramble("0123456789abcdef")
	src := []byte("payload-bytes")

	enc, err := s.Encrypt(src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(enc, src) {
		t.Fatal("ciphertext must differ from plaintext")
	}

	got, err := s.Decrypt(enc, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, src) {
		t.Fatalf("got %q want %q", got, src)
	}
}

func TestPassphraseKeyDoesNotPanic(t *testing.T) {
	s := MustNewScramble("not-an-aes-key-length")
	enc, err := s.Encrypt([]byte("x"), nil)
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.Decrypt(enc, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "x" {
		t.Fatalf("got %q", got)
	}
}

func TestDecryptShortCiphertext(t *testing.T) {
	s := MustNewScramble("0123456789abcdef")
	_, err := s.Decrypt([]byte("short"), nil)
	if err != ErrCiphertextTooShort {
		t.Fatalf("expected ErrCiphertextTooShort, got %v", err)
	}
}

func TestDecryptLegacyNonceLayout(t *testing.T) {
	s := MustNewScramble("0123456789abcdef")
	src := []byte("legacy-payload")

	nonce := make([]byte, s.nonceSize)
	copy(nonce[:4], s.ad[:])
	if _, err := rand.Read(nonce[4:]); err != nil {
		t.Fatal(err)
	}

	ct := s.aead.Seal(append([]byte{}, nonce...), nonce, src, s.ad[:])

	got, err := s.Decrypt(ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, src) {
		t.Fatalf("got %q want %q", got, src)
	}
}
