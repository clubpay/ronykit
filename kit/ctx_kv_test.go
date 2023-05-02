package kit_test

import (
	"github.com/clubpay/ronykit/kit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable(
	"GetString",
	func(key string, val, def string) {
		ctx := kit.NewContext(nil)
		Expect(ctx.GetString(key, def)).To(Equal(def))
		Expect(ctx.Set(key, val).GetString(key, def)).To(Equal(val))
	},
	Entry("", "key1", "val1", "defVal1"),
	Entry("", "key2", "val2", "defVal2"),
	Entry("", "key1", "val1", ""),
	Entry("", "key1", "", "defVal1"),
)

var _ = DescribeTable(
	"GetInt32",
	func(key string, val, def int32) {
		ctx := kit.NewContext(nil)
		Expect(ctx.GetInt32(key, def)).To(Equal(def))
		Expect(ctx.Set(key, val).GetInt32(key, def)).To(Equal(val))
	},
	Entry("", "key1", int32(100), int32(200)),
	Entry("", "key2", int32(50), int32(0)),
	Entry("", "key1", int32(0), int32(0)),
	Entry("", "key1", int32(-1), int32(10000)),
)

var _ = DescribeTable(
	"GetInt64",
	func(key string, val, def int64) {
		ctx := kit.NewContext(nil)
		Expect(ctx.GetInt64(key, def)).To(Equal(def))
		Expect(ctx.Set(key, val).GetInt64(key, def)).To(Equal(val))
	},
	Entry("", "key1", int64(100), int64(200)),
	Entry("", "key2", int64(50), int64(0)),
	Entry("", "key1", int64(0), int64(0)),
	Entry("", "key1", int64(-1), int64(10000)),
)

var _ = DescribeTable(
	"GetUInt64",
	func(key string, val, def uint64) {
		ctx := kit.NewContext(nil)
		Expect(ctx.GetUint64(key, def)).To(Equal(def))
		Expect(ctx.Set(key, val).GetUint64(key, def)).To(Equal(val))
	},
	Entry("", "key1", uint64(100), uint64(200)),
	Entry("", "key2", uint64(50), uint64(0)),
	Entry("", "key1", uint64(0), uint64(0)),
)

var _ = DescribeTable(
	"GetUInt32",
	func(key string, val, def uint32) {
		ctx := kit.NewContext(nil)
		Expect(ctx.GetUint32(key, def)).To(Equal(def))
		Expect(ctx.Set(key, val).GetUint32(key, def)).To(Equal(val))
	},
	Entry("", "key1", uint32(100), uint32(200)),
	Entry("", "key2", uint32(50), uint32(0)),
	Entry("", "key1", uint32(0), uint32(0)),
)

var _ = Describe(
	"Walk",
	func() {
		ctx := kit.NewContext(nil)
		ctx.Set("k1", "v1")
		ctx.Set("k2", "v2")

		hdr := map[string]any{}
		ctx.Walk(func(key string, val any) bool {
			hdr[key] = val

			return true
		})

		It("Should load all the header keys", func() {
			Expect(hdr).To(HaveKeyWithValue("k1", "v1"))
			Expect(hdr).To(HaveKeyWithValue("k2", "v2"))
		})
	},
)
