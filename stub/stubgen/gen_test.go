package stubgen_test

import (
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/stub/stubgen"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type dummyRESTSelector struct {
	enc    kit.Encoding
	path   string
	method string
}

func newREST(enc kit.Encoding, path, method string) dummyRESTSelector {
	return dummyRESTSelector{
		enc:    enc,
		path:   path,
		method: method,
	}
}

func (d dummyRESTSelector) Query(_ string) any {
	return nil
}

func (d dummyRESTSelector) GetEncoding() kit.Encoding {
	return d.enc
}

func (d dummyRESTSelector) GetMethod() string {
	return d.method
}

func (d dummyRESTSelector) GetPath() string {
	return d.path
}

type SimpleObject struct {
	Bool        bool    `json:"bool"`
	FloatNumber float64 `json:"floatNumber"`
}

type ComplexRequest struct {
	Str    string            `json:"str"`
	Number int64             `json:"number"`
	M      map[string]string `json:"m"`
	MI     map[string]int    `json:"mi"`
	MIS    map[int64]string  `json:"mis"`
	Arr    []SimpleObject    `json:"arr"`
	Obj    *SimpleObject     `json:"obj"`
}

type ComplexResponse struct {
	SimpleObject
	Simple  []float64        `json:"simple"`
	Complex []ComplexRequest `json:"complex"`
}

func TestStubCodeGenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stub Code Generator")
}

var _ = Describe("GolangGenerator", func() {
	It("generate dto code in golang (no comment)", func() {
		svc := desc.ServiceDescFunc(func() *desc.Service {
			return desc.NewService("testService").
				AddContract(
					desc.NewContract().
						SetName("c1").
						AddRoute(desc.Route("s1", newREST(kit.JSON, "/path1", "GET"))).
						SetInput(&ComplexRequest{}).
						SetOutput(&ComplexResponse{}).
						SetHandler(nil),
				)
		})

		in := stubgen.NewInput("test", svc)
		in.AddTags("json")
		_, err := stubgen.NewGolangEngine(stubgen.GolangConfig{PkgName: "test"}).Generate(in)
		Expect(err).To(BeNil())
	})
})

var _ = Describe("TypeScriptGenerator", func() {
	It("generate dto code in typescript (no comment)", func() {
		svc := desc.ServiceDescFunc(func() *desc.Service {
			return desc.NewService("testService").
				AddContract(
					desc.NewContract().
						SetName("c1").
						AddRoute(desc.Route("s1", newREST(kit.JSON, "/path1", "GET"))).
						SetInput(&ComplexRequest{}).
						SetOutput(&ComplexResponse{}).
						SetHandler(nil),
				)
		})

		in := stubgen.NewInput("test", svc)
		in.AddTags("json")
		in.AddExtraOptions(map[string]string{
			"withHook": "yes",
		})

		_, err := stubgen.NewTypescriptEngine(stubgen.TypescriptConfig{}).Generate(in)
		Expect(err).To(BeNil())
	})
})
