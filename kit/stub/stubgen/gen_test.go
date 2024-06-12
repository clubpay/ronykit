package stubgen

import (
	"reflect"
	"testing"

	"github.com/clubpay/ronykit/kit/desc"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStubCodeGenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stub Code Generator")
}

var _ = Describe("GolangGenerator", func() {
	It("generate dto code in golang (no comment)", func() {
		_, err := GolangStub(
			Input{
				Stub: desc.Stub{
					DTOs: map[string]desc.DTO{
						"Struct1": {
							Name:  "Struct1",
							RType: reflect.TypeOf(struct{}{}),
							Type:  "struct",
							Fields: []desc.DTOField{
								{
									Name:  "Param1",
									RType: reflect.TypeOf(*new(string)),
									Type:  "string",
									Tags: []desc.DTOFieldTag{
										{
											Name:  "json",
											Value: "param1",
										},
									},
								},
								{
									Name:  "Param2",
									RType: reflect.TypeOf(new(int)),
									Type:  "*int",
									Tags: []desc.DTOFieldTag{
										{
											Name:  "json",
											Value: "param2",
										},
									},
								},
							},
						},
					},
					RESTs: nil,
					RPCs:  nil,
				},
			},
		)
		Expect(err).To(BeNil())
	})
	It("generate dto code in golang (with comment)", func() {
		_, err := GolangStub(
			Input{
				Stub: desc.Stub{
					DTOs: map[string]desc.DTO{
						"Struct1": {
							Comments: []string{
								"Something1",
								"Something2",
							},
							Name:  "Struct1",
							RType: reflect.TypeOf(struct{}{}),
							Type:  "struct",
							Fields: []desc.DTOField{
								{
									Name:  "Param1",
									RType: reflect.TypeOf(*new(string)),
									Type:  "string",
									Tags: []desc.DTOFieldTag{
										{
											Name:  "json",
											Value: "param1",
										},
									},
								},
								{
									Name:  "Param2",
									RType: reflect.TypeOf(new(int)),
									Type:  "*int",
									Tags: []desc.DTOFieldTag{
										{
											Name:  "json",
											Value: "param2",
										},
									},
								},
							},
						},
					},
					RESTs: nil,
					RPCs:  nil,
				},
			},
		)
		Expect(err).To(BeNil())
	})
})
