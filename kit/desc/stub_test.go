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
		Expect(d.DTOs).To(HaveLen(2))
		Expect(d.DTOs["customSubStruct"].Fields).To(HaveLen(2))
		Expect(d.DTOs["customSubStruct"].Fields[0].Name).To(Equal("SubParam1"))
		Expect(d.DTOs["customSubStruct"].Fields[0].Tags).To(HaveLen(1))
		Expect(d.DTOs["customSubStruct"].Fields[1].Name).To(Equal("SubParam2"))
		Expect(d.DTOs["customSubStruct"].Fields[0].Tags).To(HaveLen(1))
		Expect(d.DTOs["customStruct"].Fields).To(HaveLen(3))
		Expect(d.DTOs["customStruct"].Fields[0].Name).To(Equal("Param1"))
		Expect(d.DTOs["customStruct"].Fields[0].Tags).To(HaveLen(1))
		Expect(d.DTOs["customStruct"].Fields[1].Name).To(Equal("Param2"))
		Expect(d.DTOs["customStruct"].Fields[0].Tags).To(HaveLen(1))
	})
})
