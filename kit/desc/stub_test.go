package desc_test

import (
	"reflect"

	"github.com/clubpay/ronykit/kit/desc"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Desc", func() {
	It("should detect all DTOs", func() {
		d := desc.NewStub("json")

		Expect(d.AddDTO(reflect.TypeOf(&customStruct{}), false)).To(Succeed())
		Expect(d.DTOs).To(HaveLen(3))
		Expect(d.DTOs["customSubStruct"].Fields).To(HaveLen(4))
		Expect(d.DTOs["customSubStruct"].Fields[0].Name).To(Equal("SubParam1"))
		Expect(d.DTOs["customSubStruct"].Fields[0].Tags).To(HaveLen(1))
		Expect(d.DTOs["customSubStruct"].Fields[1].Name).To(Equal("SubParam2"))
		Expect(d.DTOs["customSubStruct"].Fields[1].Tags).To(HaveLen(1))
		Expect(d.DTOs["customSubStruct"].Fields[2].Name).To(Equal("MapParam"))
		Expect(d.DTOs["customSubStruct"].Fields[2].SubType1).To(Equal("string"))
		Expect(d.DTOs["customSubStruct"].Fields[2].SubType2).To(Equal("anotherSubStruct"))
		Expect(d.DTOs["customSubStruct"].Fields[2].Tags).To(HaveLen(1))
		Expect(d.DTOs["customSubStruct"].Fields[3].Name).To(Equal("MapPtrParam"))
		Expect(d.DTOs["customSubStruct"].Fields[3].SubType1).To(Equal("int"))
		Expect(d.DTOs["customSubStruct"].Fields[3].SubType2).To(Equal("*anotherSubStruct"))
		Expect(d.DTOs["customSubStruct"].Fields[3].Tags).To(HaveLen(1))

		Expect(d.DTOs["customStruct"].Fields).To(HaveLen(5))
		Expect(d.DTOs["customStruct"].Fields[0].Name).To(Equal("Param1"))
		Expect(d.DTOs["customStruct"].Fields[0].Tags).To(HaveLen(1))
		Expect(d.DTOs["customStruct"].Fields[1].Name).To(Equal("Param2"))
		Expect(d.DTOs["customStruct"].Fields[0].Tags).To(HaveLen(1))
	})
})
