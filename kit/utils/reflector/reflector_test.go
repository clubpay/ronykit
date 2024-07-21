package reflector_test

import (
	"testing"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/reflector"
	goreflect "github.com/goccy/go-reflect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestReflector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reflector Suite")
}

type testMessage struct {
	testEmbed1
	X string `json:"xTag" otherTag:"xOther"`
	Y int64  `json:"yTag" otherTag:"yOther"`
	z string
	M map[string]string
	testEmbed2
}

type testEmbed1 struct {
	YY int64  `json:"yy" otherTag:"yyOther"`
	MM string `json:"mM" otherTag:"mMOther"`
}

type testEmbed2 struct {
	YYY int64  `json:"yyy" otherTag:"yyyOther"`
	MMM string `json:"mMM" otherTag:"mMMOther"`
}

var _ = Describe("Reflector", func() {
	r := reflector.New()
	m := &testMessage{
		testEmbed1: testEmbed1{
			YY: 123,
			MM: "mM",
		},
		testEmbed2: testEmbed2{
			YYY: 1234,
			MMM: "mMM",
		},
		X: "xValue",
		Y: 10,
		z: "zValue",
		M: nil,
	}
	rObj := r.Load(m, "json")

	It("Load by Struct Fields", func() {
		obj := rObj.Obj()
		Expect(obj.GetInt64(m, "YY")).To(Equal(int64(123)))
		Expect(obj.GetString(m, "MM")).To(Equal("mM"))
		Expect(obj.GetInt64(m, "YYY")).To(Equal(int64(1234)))
		Expect(obj.GetString(m, "MMM")).To(Equal("mMM"))
		Expect(obj.GetStringDefault(m, "X", "")).To(Equal(m.X))
		Expect(obj.GetInt64Default(m, "Y", 0)).To(Equal(m.Y))
		Expect(obj.GetStringDefault(m, "z", "")).To(BeEmpty())
	})

	It("Load by ToJSON tag", func() {
		byTag, ok := rObj.ByTag("json")
		Expect(ok).To(BeTrue())
		Expect(byTag.GetStringDefault(m, "xTag", "")).To(Equal(m.X))
		Expect(byTag.GetInt64Default(m, "yTag", 0)).To(Equal(m.Y))
		Expect(byTag.GetInt64Default(m, "yy", 0)).To(Equal(m.YY))
		Expect(byTag.GetInt64Default(m, "yyy", 0)).To(Equal(m.YYY))
		Expect(byTag.GetStringDefault(m, "mM", "")).To(Equal(m.MM))
		Expect(byTag.GetStringDefault(m, "mMM", "")).To(Equal(m.MMM))
		Expect(byTag.GetStringDefault(m, "z", "def")).To(Equal("def"))
	})
})

/*
Benchmark results:

cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkReflector/unsafe-16            				15217726      78.51 ns/op     0 B/op   0 allocs/op
BenchmarkReflector/unsafeRegistered-16          97216087      11.58 ns/op     0 B/op   0 allocs/op
BenchmarkReflector/reflect-16                   30267793      37.32 ns/op    16 B/op   2 allocs/op
BenchmarkReflector/ccyReflect-16                58138024      22.71 ns/op     8 B/op   1 allocs/op
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
