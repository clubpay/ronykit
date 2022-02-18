package reflector_test

import (
	"reflect"
	"testing"

	"github.com/clubpay/ronykit/reflector"
	"github.com/clubpay/ronykit/utils"
)

type testMessage struct {
	X string
	Y int64
	z string
	M map[string]string
}

func (t testMessage) Marshal() ([]byte, error) {
	return nil, nil
}

func TestReflector(t *testing.T) {
	r := reflector.New()
	m := &testMessage{
		X: "xValue",
		Y: 10,
		z: "zValue",
		M: nil,
	}
	t.Log(r.Get(m, "X"))
}

func TestExtractInfo(t *testing.T) {
	r := reflector.New()
	m := &testMessage{
		X: "xValue",
		Y: 10,
		z: "zValue",
		M: nil,
	}
	obj := r.Load(m)
	t.Log(reflect.Indirect(reflect.ValueOf(m)).Type().String())
	t.Log(obj.GetInt64("Y"))
	t.Log(obj.GetString("X"))
	t.Log(obj.GetString("Z"))
}

/*

Benchmark results:

BenchmarkReflector/unsafe-16            	18662281      60.08 ns/op    5 B/op    1 allocs/op
BenchmarkReflector/unsafeRegistered-16      77882029      15.52 ns/op    5 B/op    1 allocs/op
BenchmarkReflector/reflect-16               30601716      35.86 ns/op   24 B/op    3 allocs/op

*/
func BenchmarkReflector(b *testing.B) {
	benchs := []struct {
		name string
		f    func(*testing.B)
	}{
		{"unsafe", benchUnsafe},
		{"unsafeRegistered", benchUnsafeRegistered},
		{"reflect", benchReflect},
	}

	for idx := range benchs {
		b.ResetTimer()
		b.ReportAllocs()
		b.Run(benchs[idx].name, benchs[idx].f)
	}
}

func benchUnsafe(b *testing.B) {
	r := reflector.New()
	b.RunParallel(func(pb *testing.PB) {
		t := &testMessage{}
		for pb.Next() {
			t.X = utils.RandomID(5)

			xR, err := r.Load(t).GetString("X")
			if err != nil {
				b.Fatal(err)
			}
			if xR != t.X {
				b.Fatal(xR, t.X)
			}
		}
	})
}

func benchUnsafeRegistered(b *testing.B) {
	r := reflector.New()
	reflector.Register(&testMessage{})
	b.RunParallel(func(pb *testing.PB) {
		t := &testMessage{}
		for pb.Next() {
			t.X = utils.RandomID(5)

			xR, err := r.Load(t).GetString("X")
			if err != nil {
				b.Fatal(err)
			}
			if xR != t.X {
				b.Fatal(xR, t.X)
			}
		}
	})
}

func benchReflect(b *testing.B) {
	r := reflector.New()
	b.RunParallel(func(pb *testing.PB) {
		t := &testMessage{}
		for pb.Next() {
			t.X = utils.RandomID(5)

			xR, err := r.GetString(t, "X")
			if err != nil {
				b.Fatal(err)
			}
			if xR != t.X {
				b.Fatal(xR, t.X)
			}
		}
	})
}
