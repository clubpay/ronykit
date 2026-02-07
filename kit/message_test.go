package kit_test

import (
	"bytes"
	"reflect"

	"github.com/clubpay/ronykit/kit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type factoryMsg struct {
	A string `json:"a"`
}

var _ = Describe("MessageFactory", func() {
	It("should build factories for special messages", func() {
		nilFactory := kit.CreateMessageFactory(nil)
		Expect(nilFactory()).To(BeAssignableToTypeOf(kit.RawMessage{}))

		rawFactory := kit.CreateMessageFactory(kit.RawMessage{})
		Expect(rawFactory()).To(BeAssignableToTypeOf(kit.RawMessage{}))

		multipartFactory := kit.CreateMessageFactory(kit.MultipartFormMessage{})
		Expect(multipartFactory()).To(BeAssignableToTypeOf(kit.MultipartFormMessage{}))
	})

	It("should build factories for struct pointers", func() {
		f := kit.CreateMessageFactory(&factoryMsg{})
		msg := f()
		Expect(reflect.TypeOf(msg)).To(Equal(reflect.TypeOf(&factoryMsg{})))
	})
})

var _ = Describe("Message codec helpers", func() {
	It("should marshal and encode raw messages without JSON wrapping", func() {
		raw := kit.RawMessage("abc")
		b, err := kit.MarshalMessage(raw)
		Expect(err).To(BeNil())
		Expect(b).To(Equal([]byte("abc")))

		buf := bytes.NewBuffer(nil)
		Expect(kit.EncodeMessage(raw, buf)).To(BeNil())
		Expect(buf.String()).To(Equal("abc"))
	})

	It("should cast raw messages into structured types", func() {
		raw, err := kit.MarshalMessage(factoryMsg{A: "ok"})
		Expect(err).To(BeNil())

		out, err := kit.CastRawMessage[factoryMsg](kit.RawMessage(raw))
		Expect(err).To(BeNil())
		Expect(out.A).To(Equal("ok"))
	})

	It("should support RawMessage helpers", func() {
		var raw kit.RawMessage
		raw.CopyFrom([]byte("copy"))

		dst := make([]byte, 4)
		raw.CopyTo(dst)
		Expect(dst).To(Equal([]byte("copy")))

		cloned := raw.Clone(nil)
		Expect(cloned).To(Equal([]byte("copy")))
	})
})
