package reflector_test

import (
	"testing"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/reflector"
	goreflect "github.com/goccy/go-reflect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMessage struct {
	testEmbed1
	testEmbed2

	X string `json:"xTag" otherTag:"xOther"`
	Y int64  `json:"yTag" otherTag:"yOther"`
	z string
	M map[string]string
}

type testEmbed1 struct {
	YY int64  `json:"yy" otherTag:"yyOther"`
	MM string `json:"mM" otherTag:"mMOther"`
	Y  int64  `json:"y"`
}

type testEmbed2 struct {
	YYY int64  `json:"yyy" otherTag:"yyyOther"`
	MMM string `json:"mMM" otherTag:"mMMOther"`
	Y   int64  `json:"y2"`
}

func TestReflector(t *testing.T) {
	r := reflector.New()
	m := &testMessage{
		testEmbed1: testEmbed1{
			YY: 123,
			MM: "mM",
			Y:  11,
		},
		testEmbed2: testEmbed2{
			YYY: 1234,
			MMM: "mMM",
			Y:   12,
		},
		X: "xValue",
		Y: 10,
		z: "zValue",
		M: map[string]string{
			"x": "100",
		},
	}
	rObj := r.Load(m, "json")

	t.Run("Load by Struct Fields", func(t *testing.T) {
		assert.Equal(t, int64(10), m.Y)
		obj := rObj.Obj()

		yy, err := obj.GetInt64(m, "YY")
		require.NoError(t, err)
		assert.Equal(t, int64(123), yy)

		mm, err := obj.GetString(m, "MM")
		require.NoError(t, err)
		assert.Equal(t, "mM", mm)

		yyy, err := obj.GetInt64(m, "YYY")
		require.NoError(t, err)
		assert.Equal(t, int64(1234), yyy)

		mmm, err := obj.GetString(m, "MMM")
		require.NoError(t, err)
		assert.Equal(t, "mMM", mmm)

		assert.Equal(t, m.X, obj.GetStringDefault(m, "X", ""))
		assert.Equal(t, m.Y, obj.GetInt64Default(m, "Y", 0))
		assert.Empty(t, obj.GetStringDefault(m, "z", ""))
		assert.Equal(t, map[string]string{"x": "100"}, obj.Get(m, "M"))

		y, err := obj.GetInt64(m, "Y")
		require.NoError(t, err)
		assert.Equal(t, int64(10), y)
	})

	t.Run("Load by ToJSON tag", func(t *testing.T) {
		byTag, ok := rObj.ByTag("json")
		assert.True(t, ok)
		assert.Equal(t, int64(11), byTag.GetInt64Default(m, "y", 0))
		assert.Equal(t, m.X, byTag.GetStringDefault(m, "xTag", ""))
		assert.Equal(t, m.Y, byTag.GetInt64Default(m, "yTag", 0))
		assert.Equal(t, m.YY, byTag.GetInt64Default(m, "yy", 0))
		assert.Equal(t, m.YYY, byTag.GetInt64Default(m, "yyy", 0))
		assert.Equal(t, m.MM, byTag.GetStringDefault(m, "mM", ""))
		assert.Equal(t, m.MMM, byTag.GetStringDefault(m, "mMM", ""))
		assert.Equal(t, "def", byTag.GetStringDefault(m, "z", "def"))
	})
}

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
