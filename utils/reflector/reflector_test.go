package reflector_test

import (
	"testing"

	"github.com/clubpay/ronykit/utils"
	"github.com/clubpay/ronykit/utils/reflector"
	goreflect "github.com/goccy/go-reflect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestReflector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reflector Suite")
}

type testMessage struct {
	X string `json:"xTag" otherTag:"xOther"`
	Y int64  `json:"yTag" otherTag:"yOther"`
	z string
	M map[string]string
}

var _ = Describe("Reflector", func() {
	r := reflector.New()
	m := &testMessage{
		X: "xValue",
		Y: 10,
		z: "zValue",
		M: nil,
	}
	rObj := r.Load(m, "json")

	It("Load by Struct Fields", func() {
		obj := rObj.Obj()
		Expect(obj.GetStringDefault(m, "X", "")).To(Equal(m.X))
		Expect(obj.GetInt64Default(m, "Y", 0)).To(Equal(m.Y))
		Expect(obj.GetStringDefault(m, "z", "")).To(BeEmpty())
	})

	It("Load by JSON tag", func() {
		byTag, ok := rObj.ByTag("json")
		Expect(ok).To(BeTrue())
		Expect(byTag.GetStringDefault(m, "xTag", "")).To(Equal(m.X))
		Expect(byTag.GetInt64Default(m, "yTag", 0)).To(Equal(m.Y))
		Expect(byTag.GetStringDefault(m, "z", "def")).To(Equal("def"))
	})
})

/*
Benchmark results:

BenchmarkReflector/unsafe-16            	18662281      60.08 ns/op    5 B/op    1 allocs/op.
BenchmarkReflector/unsafeRegistered-16      77882029      15.52 ns/op    5 B/op    1 allocs/op.
BenchmarkReflector/reflect-16               30601716      35.86 ns/op   24 B/op    3 allocs/op.
*/
func BenchmarkReflector(b *testing.B) {
	benchs := []struct {
		name string
		f    func(*testing.B)
	}{
		{"unsafe", benchUnsafe},
		{"unsafeRegistered", benchUnsafeRegistered},
		{"reflect", benchReflect},
		{"ccyReflect", benchGoReflect},
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
		t.X = utils.RandomID(5)
		for pb.Next() {
			obj := r.Load(t).Obj()
			xR, err := obj.GetString(t, "X")
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
		t.X = utils.RandomID(5)
		for pb.Next() {
			obj := r.Load(t).Obj()
			xR, err := obj.GetString(t, "X")
			if err != nil {
				b.Fatal(err)
			}
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
		t.X = utils.RandomID(5)
		for pb.Next() {
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

func benchGoReflect(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		t := &testMessage{}
		t.X = utils.RandomID(5)
		for pb.Next() {
			xR := goreflect.Indirect(goreflect.ValueOf(t)).FieldByName("X").String()
			if xR != t.X {
				b.Fatal(xR, t.X)
			}
		}
	})
}
