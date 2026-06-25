package kit_test

import (
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/stretchr/testify/assert"
)

func TestGetString(t *testing.T) {
	tests := []struct {
		key string
		val string
		def string
	}{
		{"key1", "val1", "defVal1"},
		{"key2", "val2", "defVal2"},
		{"key1", "val1", ""},
		{"key1", "", "defVal1"},
	}
	for _, tt := range tests {
		t.Run(tt.key+"_"+tt.val+"_"+tt.def, func(t *testing.T) {
			ctx := kit.NewContext(nil)
			assert.Equal(t, tt.def, ctx.GetString(tt.key, tt.def))
			assert.Equal(t, tt.val, ctx.Set(tt.key, tt.val).GetString(tt.key, tt.def))
		})
	}
}

func TestGetInt32(t *testing.T) {
	tests := []struct {
		key string
		val int32
		def int32
	}{
		{"key1", 100, 200},
		{"key2", 50, 0},
		{"key1", 0, 0},
		{"key1", -1, 10000},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			ctx := kit.NewContext(nil)
			assert.Equal(t, tt.def, ctx.GetInt32(tt.key, tt.def))
			assert.Equal(t, tt.val, ctx.Set(tt.key, tt.val).GetInt32(tt.key, tt.def))
		})
	}
}

func TestGetInt64(t *testing.T) {
	tests := []struct {
		key string
		val int64
		def int64
	}{
		{"key1", 100, 200},
		{"key2", 50, 0},
		{"key1", 0, 0},
		{"key1", -1, 10000},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			ctx := kit.NewContext(nil)
			assert.Equal(t, tt.def, ctx.GetInt64(tt.key, tt.def))
			assert.Equal(t, tt.val, ctx.Set(tt.key, tt.val).GetInt64(tt.key, tt.def))
		})
	}
}

func TestGetUint64(t *testing.T) {
	tests := []struct {
		key string
		val uint64
		def uint64
	}{
		{"key1", 100, 200},
		{"key2", 50, 0},
		{"key1", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			ctx := kit.NewContext(nil)
			assert.Equal(t, tt.def, ctx.GetUint64(tt.key, tt.def))
			assert.Equal(t, tt.val, ctx.Set(tt.key, tt.val).GetUint64(tt.key, tt.def))
		})
	}
}

func TestGetUint32(t *testing.T) {
	tests := []struct {
		key string
		val uint32
		def uint32
	}{
		{"key1", 100, 200},
		{"key2", 50, 0},
		{"key1", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			ctx := kit.NewContext(nil)
			assert.Equal(t, tt.def, ctx.GetUint32(tt.key, tt.def))
			assert.Equal(t, tt.val, ctx.Set(tt.key, tt.val).GetUint32(tt.key, tt.def))
		})
	}
}

func TestWalk(t *testing.T) {
	ctx := kit.NewContext(nil)
	ctx.Set("k1", "v1")
	ctx.Set("k2", "v2")

	hdr := map[string]any{}
	ctx.Walk(func(key string, val any) bool {
		hdr[key] = val

		return true
	})

	assert.Equal(t, "v1", hdr["k1"])
	assert.Equal(t, "v2", hdr["k2"])
}
