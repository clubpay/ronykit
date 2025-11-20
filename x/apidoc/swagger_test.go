package apidoc_test

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/clubpay/ronykit/contrib/swagger"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/clubpay/ronykit/x/apidoc"
	"github.com/clubpay/ronykit/x/apidoc/internal/testdata/a"
	"github.com/clubpay/ronykit/x/apidoc/internal/testdata/b"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type embedded struct {
	Embedded string `json:"embedded"`
	Others   string `json:"others"`
}

type sampleReq struct {
	embedded
	TT  [][]string        `json:"tt"`
	X   string            `json:"x,int" swag:"enum:1,2,3"`
	Y   string            `json:"y,omitempty"`
	Z   int64             `json:"z" swag:"deprecated;optional"`
	W   []string          `json:"w"`
	MS  map[string]subRes `json:"ms"`
	MSS map[string]string `json:"mss"`
	MIS map[int]string    `json:"mis"`
	MII map[int]int       `json:"mii"`
	CT  time.Time         `json:"ct"`
	CTP *time.Time        `json:"ctp"`
}

type subRes struct {
	Some    string `json:"some"`
	Another []byte `json:"another"`
}

type sampleRes struct {
	Out1  int       `json:"out1"`
	Out2  string    `json:"out2"`
	Sub   subRes    `json:"sub"`
	Subs  []subRes  `json:"subs,omitempty"`
	XSub  *subRes   `json:"xSub"`
	Subs2 []*subRes `json:"subs2,omitempty"`
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
				AddRoute(desc.Route("", fasthttp.GET("/some/:x/:y"))).
				SetInput(&sampleReq{}).
				SetOutput(&sampleRes{}).
				AddError(&sampleError{404, "ITEM1"}).
				AddError(&sampleError{504, "SERVER"}).
				SetHandler(nil),
		).
		AddContract(
			desc.NewContract().
				SetName("sumPOST").
				AddRoute(desc.Route("", fasthttp.POST("/some/:x/:y"))).
				SetInput(&sampleReq{}).
				SetOutput(&anotherRes{}).
				SetHandler(nil),
		).
		AddContract(
			desc.NewContract().
				SetName("sumPOST2").
				AddRoute(desc.Route("", fasthttp.POST("/some/something-else"))).
				SetInput(kit.RawMessage{}).
				SetOutput(kit.RawMessage{}).
				SetHandler(nil),
		).
		AddContract(
			desc.NewContract().
				SetName("upload").
				AddRoute(desc.Route("", fasthttp.POST("/upload"))).
				SetInput(kit.MultipartFormMessage{},
					desc.WithField("file", desc.FieldMeta{
						FormData: &desc.FormDataValue{Name: "file", Type: "file"},
					}),
				).
				SetOutput(kit.RawMessage{}).
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
	ps2 := desc.Parse(testService2{})
	It("Check Request for Contract[0].Request (sampleReq)", func() {
		d := apidoc.ToSwaggerDefinition(ps.Origin.Name, ps.Contracts[0].Request.Message)
		props := d.Properties.ToOrderedSchemaItems()
		Expect(props[0].Name).To(Equal("others"))
		Expect(props[0].SchemaProps.Type[0]).To(Equal("string"))
		Expect(props[1].Name).To(Equal("embedded"))
		Expect(props[1].SchemaProps.Type[0]).To(Equal("string"))
		Expect(props[2].Name).To(Equal("tt"))
		Expect(props[2].SchemaProps.Type[0]).To(Equal("array"))
		Expect(props[2].SchemaProps.Items.Schema.Type[0]).To(Equal("array"))
		Expect(props[2].SchemaProps.Items.Schema.Items.Schema.Type[0]).To(Equal("string"))
		Expect(props[3].Name).To(Equal("x"))
		Expect(props[3].SchemaProps.Type[0]).To(Equal("string"))
		Expect(props[4].Name).To(Equal("y"))
		Expect(props[4].SchemaProps.Type[0]).To(Equal("string"))
		Expect(props[5].Name).To(Equal("z"))
		Expect(props[5].SchemaProps.Type[0]).To(Equal("integer"))
		Expect(props[5].SchemaProps.Description).To(Equal("[Deprecated] [Optional]"))
		Expect(props[6].Name).To(Equal("w"))
		Expect(props[6].SchemaProps.Type[0]).To(Equal("array"))
		Expect(props[6].SchemaProps.Items.Schema.Type[0]).To(Equal("string"))
	})

	It("Check Request for Contract[0].Response (sampleRes)", func() {
		d := apidoc.ToSwaggerDefinition(ps.Origin.Name, ps.Contracts[0].Responses[0].Message)
		props := d.Properties.ToOrderedSchemaItems()
		Expect(props[0].Name).To(Equal("out1"))
		Expect(props[0].SchemaProps.Type[0]).To(Equal("integer"))
		Expect(props[1].Name).To(Equal("out2"))
		Expect(props[1].SchemaProps.Type[0]).To(Equal("string"))
		Expect(props[2].Name).To(Equal("sub"))
		Expect(props[2].SchemaProps.Ref.String()).To(Equal("#/definitions/testService.subRes"))
		Expect(props[3].Name).To(Equal("subs"))
		Expect(props[3].SchemaProps.Type[0]).To(Equal("array"))
		Expect(props[3].SchemaProps.Items.Schema.Ref.String()).To(Equal("#/definitions/testService.subRes"))
		Expect(props[4].Name).To(Equal("xSub"))
		Expect(props[4].SchemaProps.Ref.String()).To(Equal("#/definitions/testService.subRes"))
		Expect(props[4].SchemaProps.Description).To(Equal("[Optional]"))
		Expect(props[5].Name).To(Equal("subs2"))
		Expect(props[5].SchemaProps.Type[0]).To(Equal("array"))
		Expect(props[5].SchemaProps.Items.Schema.Ref.String()).To(Equal("#/definitions/testService.subRes"))
	})

	It("Check Request for Contract[1].Response (anotherRes)", func() {
		d := apidoc.ToSwaggerDefinition(ps.Origin.Name, ps.Contracts[1].Responses[0].Message)
		props := d.Properties.ToOrderedSchemaItems()
		Expect(props[0].Name).To(Equal("another"))
		Expect(props[0].SchemaProps.Type[0]).To(Equal("string"))
		Expect(props[1].Name).To(Equal("some"))
		Expect(props[1].SchemaProps.Type[0]).To(Equal("string"))
		Expect(props[2].Name).To(Equal("out1"))
		Expect(props[2].SchemaProps.Type[0]).To(Equal("integer"))
		Expect(props[2].Description).To(Equal("[Deprecated]"))
		Expect(props[3].Name).To(Equal("out2"))
		Expect(props[3].SchemaProps.Type[0]).To(Equal("string"))
		Expect(props[3].Enum).To(Equal([]interface{}{"OUT1", "OUT2"}))
	})

	It("Check Request for Contract[2].Response (kit.RawMessage)", func() {
		d := apidoc.ToSwaggerDefinition(ps.Origin.Name, ps.Contracts[2].Responses[0].Message)
		props := d.Properties.ToOrderedSchemaItems()

		_ = props
	})

	It("Check Request for Contract[3] (kit.MultipartForm)", func() {
		d := apidoc.ToSwaggerDefinition(ps.Origin.Name, ps.Contracts[3].Responses[0].Message)
		props := d.Properties.ToOrderedSchemaItems()

		_ = props
	})

	It("Name Collision from Multiple Package", func() {
		Expect(ps2.Messages()).To(HaveLen(3))
		fmt.Println(ps2.Messages())
		err := swagger.New("Test2", "", "").WriteSwagToFile("_swagger2.json", testService2{})
		Expect(err).To(BeNil())
	})
})

func BenchmarkSwagger(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ps := desc.Parse(testService{})
		ps.Messages()
		err := swagger.New("title", "v1.0.0", "").
			WriteSwagTo(io.Discard, testService{})
		if err != nil {
			b.Error(err)
		}
	}
}

type testService2 struct{}

type request struct {
	A a.Document `json:"a"`
	B b.Document `json:"b"`
}

func (t testService2) Desc() *desc.Service {
	return (&desc.Service{
		Name: "testService",
	}).
		AddContract(
			desc.NewContract().
				SetName("sumGET").
				AddRoute(desc.Route("", fasthttp.GET("/some/:x/:y"))).
				SetInput(&request{}).
				SetOutput(&b.Document{}).
				SetHandler(nil),
		)
}
