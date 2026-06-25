package kit_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type factoryMsg struct {
	A string `json:"a"`
}

func TestMessageFactory(t *testing.T) {
	t.Run("should build factories for special messages", func(t *testing.T) {
		nilFactory := kit.CreateMessageFactory(nil)
		assert.IsType(t, kit.RawMessage{}, nilFactory())

		rawFactory := kit.CreateMessageFactory(kit.RawMessage{})
		assert.IsType(t, kit.RawMessage{}, rawFactory())

		multipartFactory := kit.CreateMessageFactory(kit.MultipartFormMessage{})
		assert.IsType(t, kit.MultipartFormMessage{}, multipartFactory())
	})

	t.Run("should build factories for struct pointers", func(t *testing.T) {
		f := kit.CreateMessageFactory(&factoryMsg{})
		msg := f()
		assert.Equal(t, reflect.TypeOf(&factoryMsg{}), reflect.TypeOf(msg))
	})
}

func TestMessageCodecHelpers(t *testing.T) {
	t.Run("should marshal and encode raw messages without JSON wrapping", func(t *testing.T) {
		raw := kit.RawMessage("abc")
		b, err := kit.MarshalMessage(raw)
		require.NoError(t, err)
		assert.Equal(t, []byte("abc"), b)

		buf := bytes.NewBuffer(nil)
		require.NoError(t, kit.EncodeMessage(raw, buf))
		assert.Equal(t, "abc", buf.String())
	})

	t.Run("should cast raw messages into structured types", func(t *testing.T) {
		raw, err := kit.MarshalMessage(factoryMsg{A: "ok"})
		require.NoError(t, err)

		out, err := kit.CastRawMessage[factoryMsg](kit.RawMessage(raw))
		require.NoError(t, err)
		assert.Equal(t, "ok", out.A)
	})

	t.Run("should support RawMessage helpers", func(t *testing.T) {
		var raw kit.RawMessage
		raw.CopyFrom([]byte("copy"))

		dst := make([]byte, 4)
		raw.CopyTo(dst)
		assert.Equal(t, []byte("copy"), dst)

		cloned := raw.Clone(nil)
		assert.Equal(t, []byte("copy"), cloned)
	})
}
