package buf_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/buf"
)

func TestBytes(t *testing.T) {
	b1 := buf.GetLen(100)
	b2 := buf.GetLen(100)
	b3 := buf.GetCap(100)
	src := utils.S2B(utils.RandomID(100))
	b1.CopyFrom(src)
	if !bytes.Equal(*b1.Bytes(), src) {
		t.Errorf("bytes not equal")
	}
	n, err := io.Copy(b2, b1)
	if err != nil {
		t.Error(err)
	}
	if n != 100 {
		t.Errorf("copied bytes not equal")
	}
	if b2.Len() != 200 {
		t.Errorf("copied bytes not equal: %d", b2.Len())
	}

	n, err = io.Copy(b3, b1)
	if err != nil {
		t.Error(err)
	}
	if n != 0 {
		t.Errorf("copied bytes not equal: %d", n)
	}
	if b3.Len() != 0 {
		t.Errorf("copied bytes not equal: %d", b3.Len())
	}

	b1.Reset()
	b1.AppendFrom(src)
	n, err = io.Copy(b3, b1)
	if err != nil {
		t.Error(err)
	}
	if n != 100 {
		t.Errorf("copied bytes not equal: %d", n)
	}
	if b3.Len() != 100 {
		t.Errorf("copied bytes not equal: %d", b3.Len())
	}
	if !bytes.Equal(*b3.Bytes(), src) {
		t.Errorf("bytes not equal")
	}
}

func BenchmarkBytesPool(b *testing.B) {
	bp := buf.NewBytesPool(32, 4098)
	data := utils.S2B(utils.RandomID(512))
	for i := 0; i < b.N; i++ {
		bb := bp.GetCap(1024)
		_, err := bb.Write(data)
		if err != nil {
			b.Fatal(err)
		}
		bb.Release()
	}
}
