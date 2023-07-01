package swagger_test

import (
	"fmt"
	"testing"

	"github.com/clubpay/ronykit/contrib/swagger"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type sampleReq struct {
	X  string     `json:"x,int"`
	Y  string     `json:"y,omitempty"`
	Z  int64      `json:"z" swag:"deprecated;optional"`
	W  []string   `json:"w"`
	TT [][]string `json:"tt"`
}

type subRes struct {
	Some    string `json:"some"`
	Another []byte `json:"another"`
}

type sampleRes struct {
	Out1 int      `json:"out1"`
	Out2 string   `json:"out2"`
	Sub  subRes   `json:"sub"`
	Subs []subRes `json:"subs"`
	XSub *subRes  `json:"xSub"`
}

type sampleError struct {
	Code int    `json:"code" swag:"enum:504,503"`
	Item string `json:"item"`
}

var _ kit.ErrorMessage = (*sampleError)(nil)

func (e sampleError) GetCode() int {
	return e.Code
}

func (e sampleError) GetItem() string {
	return e.Item
}

func (e sampleError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Item)
}

type anotherRes struct {
	subRes
	Out1 int    `json:"out1" swag:"deprecated"`
	Out2 string `json:"out2" swag:"enum:OUT1,OUT2"`
}

type testService struct{}

func (t testService) Desc() *desc.Service {
	return (&desc.Service{
		Name: "testService",
	}).
		AddContract(
			desc.NewContract().
				SetName("sumGET").
				AddSelector(fasthttp.Selector{
					Method: fasthttp.MethodGet,
					Path:   "/some/:x/:y",
				}).
				SetInput(&sampleReq{}).
				SetOutput(&sampleRes{}).
				AddError(&sampleError{404, "ITEM1"}).
				AddError(&sampleError{504, "SERVER"}).
				SetHandler(nil),
		).
		AddContract(
			desc.NewContract().
				SetName("sumPOST").
				AddSelector(fasthttp.Selector{
					Method: fasthttp.MethodPost,
					Path:   "/some/:x/:y",
				}).
				SetInput(&sampleReq{}).
				SetOutput(&anotherRes{}).
				SetHandler(nil),
		)
}

func ExampleGenerator_WriteSwagToFile() {
	err := swagger.New("TestTitle", "v0.0.1", "").
		WithTag("json").
		WriteSwagToFile("_swagger.json", testService{})
	fmt.Println(err)
	// Output: <nil>
}

func ExampleGenerator_WritePostmanToFile() {
	err := swagger.New("TestTitle", "v0.0.1", "").
		WithTag("json").
		WritePostmanToFile("_postman.json", testService{})
	fmt.Println(err)
	// Output: <nil>
}

func TestSwagger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Swagger Suite")
}

var _ = Describe("ToSwaggerDefinition", func() {
	ps := desc.Parse(testService{})
	It("Check Request for Contract[0].Request (sampleReq)", func() {
		// type sampleReq struct {
		//  	X  string     `json:"x,int"`
		//  	Y  string     `json:"y,omitempty"`
		//  	Z  int64      `json:"z"`
		//  	W  []string   `json:"w"`
		//  	TT [][]string `json:"tt"`
		// }
		// type sampleRes struct {
		// 		Out1 int      `json:"out1"`
		// 		Out2 string   `json:"out2"`
		// 		Sub  subRes   `json:"sub"`
		// 		Subs []subRes `json:"subs"`
		// 		XSub *subRes  `json:"xSub"`
		// }

		d := swagger.ToSwaggerDefinition(ps.Contracts[0].Request.Message)
		props := d.Properties.ToOrderedSchemaItems()
		Expect(props[0].Name).To(Equal("tt"))
		Expect(props[0].SchemaProps.Type[0]).To(Equal("array"))
		Expect(props[0].SchemaProps.Items.Schema.Type[0]).To(Equal("array"))
		Expect(props[0].SchemaProps.Items.Schema.Items.Schema.Type[0]).To(Equal("string"))
		Expect(props[1].Name).To(Equal("w"))
		Expect(props[1].SchemaProps.Type[0]).To(Equal("array"))
		Expect(props[1].SchemaProps.Items.Schema.Type[0]).To(Equal("string"))
		Expect(props[2].Name).To(Equal("x"))
		Expect(props[2].SchemaProps.Type[0]).To(Equal("string"))
		Expect(props[3].Name).To(Equal("y"))
		Expect(props[3].SchemaProps.Type[0]).To(Equal("string"))
		Expect(props[4].Name).To(Equal("z"))
		Expect(props[4].SchemaProps.Type[0]).To(Equal("integer"))
		Expect(props[4].SchemaProps.Description).To(Equal("[Deprecated] [Optional]"))
	})

	It("Check Request for Contract[0].Response (sampleRes)", func() {
		d := swagger.ToSwaggerDefinition(ps.Contracts[0].Responses[0].Message)
		props := d.Properties.ToOrderedSchemaItems()
		Expect(props[0].Name).To(Equal("out1"))
		Expect(props[0].SchemaProps.Type[0]).To(Equal("integer"))
		Expect(props[1].Name).To(Equal("out2"))
		Expect(props[1].SchemaProps.Type[0]).To(Equal("string"))
		Expect(props[2].Name).To(Equal("sub"))
		Expect(props[2].SchemaProps.Ref.String()).To(Equal("#/definitions/subRes"))
		Expect(props[3].Name).To(Equal("subs"))
		Expect(props[3].SchemaProps.Type[0]).To(Equal("array"))
		Expect(props[3].SchemaProps.Items.Schema.Ref.String()).To(Equal("#/definitions/subRes"))
		Expect(props[4].Name).To(Equal("xSub"))
		Expect(props[4].SchemaProps.Ref.String()).To(Equal("#/definitions/subRes"))
		Expect(props[4].SchemaProps.Description).To(Equal("[Optional]"))
	})

	It("Check Request for Contract[1].Response (anotherRes)", func() {
		// type subRes struct {
		// 	Some    string `json:"some"`
		// 	Another []byte `json:"another"`
		// }
		// type anotherRes struct {
		// 	subRes
		// 	Out1 int    `json:"out1"`
		// 	Out2 string `json:"out2"`
		// }

		d := swagger.ToSwaggerDefinition(ps.Contracts[1].Responses[0].Message)
		props := d.Properties.ToOrderedSchemaItems()
		Expect(props[0].Name).To(Equal("another"))
		Expect(props[0].SchemaProps.Type[0]).To(Equal("array"))
		Expect(props[0].SchemaProps.Items.Schema.Type[0]).To(Equal("integer"))
		Expect(props[1].Name).To(Equal("out1"))
		Expect(props[1].SchemaProps.Type[0]).To(Equal("integer"))
		Expect(props[1].Description).To(Equal("[Deprecated]"))
		Expect(props[2].Name).To(Equal("out2"))
		Expect(props[2].SchemaProps.Type[0]).To(Equal("string"))
		Expect(props[2].Enum).To(Equal([]interface{}{"OUT1", "OUT2"}))
		Expect(props[3].Name).To(Equal("some"))
		Expect(props[3].SchemaProps.Type[0]).To(Equal("string"))
	})
})
