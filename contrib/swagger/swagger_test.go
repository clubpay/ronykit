package swagger_test

import (
	"fmt"

	"github.com/clubpay/ronykit/contrib/swagger"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

type sampleReq struct {
	X  string     `json:"x,int"`
	Y  string     `json:"y,omitempty"`
	Z  int64      `json:"z"`
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
	Out1 int    `json:"out1"`
	Out2 string `json:"out2"`
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
