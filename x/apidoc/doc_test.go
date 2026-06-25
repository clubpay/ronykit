package apidoc_test

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/clubpay/ronykit/x/apidoc"
	"github.com/clubpay/ronykit/x/apidoc/internal/testdata/a"
	"github.com/clubpay/ronykit/x/apidoc/internal/testdata/b"
	"github.com/stretchr/testify/assert"
)

type embedded struct {
	Embedded string `json:"embedded"`
	Others   string `json:"others"`
}

type iErr interface {
	ErrorDetails()
}

type sampleReq struct {
	embedded
	TT     [][]string        `json:"tt"`
	X      string            `json:"x,int" swag:"enum:1,2,3"`
	Y      string            `json:"y,omitempty"`
	Z      int64             `json:"z" swag:"deprecated;optional"`
	W      []string          `json:"w"`
	MS     map[string]subRes `json:"ms"`
	MSS    map[string]string `json:"mss"`
	MIS    map[int]string    `json:"mis"`
	MII    map[int]int       `json:"mii"`
	MIA    map[string]any    `json:"mia"`
	CT     time.Time         `json:"ct"`
	CTP    *time.Time        `json:"ctp"`
	ErrMsg iErr              `json:"err"`
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
	err := apidoc.New("TestTitle", "v0.0.1", "").
		WithTag("json").
		WriteSwagToFile("_swagger.json", testService{})
	fmt.Println(err)
	// Output: <nil>
}

func ExampleGenerator_WritePostmanToFile() {
	err := apidoc.New("TestTitle", "v0.0.1", "").
		WithTag("json").
		WritePostmanToFile("_postman.json", testService{})
	fmt.Println(err)
	// Output: <nil>
}

func TestToSwaggerDefinition(t *testing.T) {
	ps := desc.Parse(testService{})
	ps2 := desc.Parse(testService2{})

	t.Run("Check Request for Contract[0].Request (sampleReq)", func(t *testing.T) {
		d := apidoc.ToSwaggerDefinition(ps.Origin.Name, ps.Contracts[0].Request.Message)
		props := d.Properties.ToOrderedSchemaItems()
		assert.Equal(t, "others", props[0].Name)
		assert.Equal(t, "string", props[0].SchemaProps.Type[0])
		assert.Equal(t, "embedded", props[1].Name)
		assert.Equal(t, "string", props[1].SchemaProps.Type[0])
		assert.Equal(t, "tt", props[2].Name)
		assert.Equal(t, "array", props[2].SchemaProps.Type[0])
		assert.Equal(t, "array", props[2].SchemaProps.Items.Schema.Type[0])
		assert.Equal(t, "string", props[2].SchemaProps.Items.Schema.Items.Schema.Type[0])
		assert.Equal(t, "x", props[3].Name)
		assert.Equal(t, "string", props[3].SchemaProps.Type[0])
		assert.Equal(t, "y", props[4].Name)
		assert.Equal(t, "string", props[4].SchemaProps.Type[0])
		assert.Equal(t, "z", props[5].Name)
		assert.Equal(t, "integer", props[5].SchemaProps.Type[0])
		assert.Equal(t, "[Deprecated] [Optional]", props[5].SchemaProps.Description)
		assert.Equal(t, "w", props[6].Name)
		assert.Equal(t, "array", props[6].SchemaProps.Type[0])
		assert.Equal(t, "string", props[6].SchemaProps.Items.Schema.Type[0])
		assert.Equal(t, "ms", props[7].Name)
		assert.Equal(t, "object", props[7].SchemaProps.Type[0])
		assert.Equal(t, "#/definitions/testService.subRes", props[7].SchemaProps.AdditionalProperties.Schema.Ref.String())
		assert.Equal(t, "mss", props[8].Name)
		assert.Equal(t, "object", props[8].SchemaProps.Type[0])
		assert.Equal(t, "mis", props[9].Name)
		assert.Equal(t, "object", props[9].SchemaProps.Type[0])
		assert.Equal(t, "mii", props[10].Name)
		assert.Equal(t, "object", props[10].SchemaProps.Type[0])
		assert.Equal(t, "mia", props[11].Name)
		assert.Equal(t, "object", props[11].SchemaProps.Type[0])
	})

	t.Run("Check Request for Contract[0].Response (sampleRes)", func(t *testing.T) {
		d := apidoc.ToSwaggerDefinition(ps.Origin.Name, ps.Contracts[0].Responses[0].Message)
		props := d.Properties.ToOrderedSchemaItems()
		assert.Equal(t, "out1", props[0].Name)
		assert.Equal(t, "integer", props[0].SchemaProps.Type[0])
		assert.Equal(t, "out2", props[1].Name)
		assert.Equal(t, "string", props[1].SchemaProps.Type[0])
		assert.Equal(t, "sub", props[2].Name)
		assert.Equal(t, "#/definitions/testService.subRes", props[2].SchemaProps.Ref.String())
		assert.Equal(t, "subs", props[3].Name)
		assert.Equal(t, "array", props[3].SchemaProps.Type[0])
		assert.Equal(t, "#/definitions/testService.subRes", props[3].SchemaProps.Items.Schema.Ref.String())
		assert.Equal(t, "xSub", props[4].Name)
		assert.Equal(t, "#/definitions/testService.subRes", props[4].SchemaProps.Ref.String())
		assert.Equal(t, "[Optional]", props[4].SchemaProps.Description)
		assert.Equal(t, "subs2", props[5].Name)
		assert.Equal(t, "array", props[5].SchemaProps.Type[0])
		assert.Equal(t, "#/definitions/testService.subRes", props[5].SchemaProps.Items.Schema.Ref.String())
	})

	t.Run("Check Request for Contract[1].Response (anotherRes)", func(t *testing.T) {
		d := apidoc.ToSwaggerDefinition(ps.Origin.Name, ps.Contracts[1].Responses[0].Message)
		props := d.Properties.ToOrderedSchemaItems()
		assert.Equal(t, "another", props[0].Name)
		assert.Equal(t, "string", props[0].SchemaProps.Type[0])
		assert.Equal(t, "some", props[1].Name)
		assert.Equal(t, "string", props[1].SchemaProps.Type[0])
		assert.Equal(t, "out1", props[2].Name)
		assert.Equal(t, "integer", props[2].SchemaProps.Type[0])
		assert.Equal(t, "[Deprecated]", props[2].Description)
		assert.Equal(t, "out2", props[3].Name)
		assert.Equal(t, "string", props[3].SchemaProps.Type[0])
		assert.Equal(t, []interface{}{"OUT1", "OUT2"}, props[3].Enum)
	})

	t.Run("Check Request for Contract[2].Response (kit.RawMessage)", func(t *testing.T) {
		d := apidoc.ToSwaggerDefinition(ps.Origin.Name, ps.Contracts[2].Responses[0].Message)
		props := d.Properties.ToOrderedSchemaItems()

		_ = props
	})

	t.Run("Check Request for Contract[3] (kit.MultipartForm)", func(t *testing.T) {
		d := apidoc.ToSwaggerDefinition(ps.Origin.Name, ps.Contracts[3].Responses[0].Message)
		props := d.Properties.ToOrderedSchemaItems()

		_ = props
	})

	t.Run("Name Collision from Multiple Package", func(t *testing.T) {
		assert.Len(t, ps2.Messages(), 3)
		fmt.Println(ps2.Messages())
		err := apidoc.New("Test2", "", "").WriteSwagToFile("_swagger2.json", testService2{})
		assert.NoError(t, err)
	})
}

func BenchmarkSwagger(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ps := desc.Parse(testService{})
		ps.Messages()
		err := apidoc.New("title", "v1.0.0", "").
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
