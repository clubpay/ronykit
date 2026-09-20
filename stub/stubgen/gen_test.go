package stubgen_test

import (
	"fmt"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/rony/errs"
	"github.com/clubpay/ronykit/stub/stubgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sampleInterface interface {
	Print() string
}

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

func (d dummyRESTSelector) String() string {
	return d.method + " " + d.path
}

type SimpleObject struct {
	Bool        bool            `json:"bool"`
	FloatNumber float64         `json:"floatNumber"`
	Printer     sampleInterface `json:"printer"`
	Error       *errs.Error     `json:"err"`
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

func TestGolangGenerator(t *testing.T) {
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
	files, err := stubgen.NewGolangEngine(stubgen.GolangConfig{PkgName: "test"}).Generate(in)
	require.NoError(t, err)
	assert.NotNil(t, files)
	assert.Greater(t, len(files), 0)

	src := string(files[0].Data)
	assert.Contains(t, src, "func (s testStub) Client() *stub.Stub")
	assert.NotContains(t, src, "hostPort  string")
	assert.NotContains(t, src, "verifyTLS bool")
	assert.Contains(t, src, "sm.s1 = f")
	assert.NotContains(t, src, `swag:""`)
	assert.Contains(t, src, "// s1 GET /path1")
	fmt.Println(src)
}

func TestTypeScriptGenerator(t *testing.T) {
	svc := desc.ServiceDescFunc(func() *desc.Service {
		return desc.NewService("testService").
			AddContract(
				desc.NewContract().
					SetName("c1").
					AddRoute(desc.Route("s1", newREST(kit.JSON, "/path1/{str}", "GET"))).
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

	files, err := stubgen.NewTypescriptEngine(stubgen.TypescriptConfig{}).Generate(in)
	require.NoError(t, err)
	require.NotEmpty(t, files)

	src := string(files[0].Data)
	assert.Contains(t, src, "export class testStubError extends globalThis.Error")
	assert.Contains(t, src, "async s1(")
	assert.Contains(t, src, "init?: RequestInit")
	assert.Contains(t, src, "encodeURIComponent")
	assert.Contains(t, src, "this.buildQuery")
	assert.Contains(t, src, "res.status !== 201")
	assert.NotContains(t, src, "@ts-ignore")
}

func TestTypeScriptSWRHooksGenerator(t *testing.T) {
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
	files, err := stubgen.NewTypescriptEngine(stubgen.TypescriptConfig{GenerateSWR: true}).Generate(in)
	require.NoError(t, err)
	require.Len(t, files, 2)
	assert.Equal(t, "swr.hooks.ts", files[1].Filename)

	src := string(files[1].Data)
	assert.Contains(t, src, "export function uses1(")
	assert.Contains(t, src, "init?: RequestInit")
	assert.Contains(t, src, "ComplexRequest")
	assert.Contains(t, src, "ComplexResponse")
	assert.NotContains(t, src, "SimpleObject")
	assert.NotContains(t, src, "@ts-ignore")
}
