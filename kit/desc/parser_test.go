package desc_test

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
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

func (d dummyRESTSelector) Query(q string) interface{} {
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

type FlatMessage struct {
	A string            `json:"a"`
	B int64             `json:"b"`
	C map[string]string `json:"c"`
	D map[string]int64  `json:"d"`
	E map[int64]string  `json:"e"`
}

type NestedMessage struct {
	A  string                 `json:"a"`
	B  FlatMessage            `json:"b"`
	BA []FlatMessage          `json:"ba"`
	BM map[string]FlatMessage `json:"bm"`
}

var _ = Describe("DescParser", func() {
	d := desc.NewService("sample").
		AddContract(
			desc.NewContract().
				SetName("c1").
				NamedSelector("s1", newREST(kit.JSON, "/path1", "GET")).
				NamedSelector("s2", newREST(kit.JSON, "/path2", "POST")).
				In(NestedMessage{}).
				Out(FlatMessage{}),
		)

	It("should parse the description", func() {
		pd := desc.ParseService(d)
		Expect(pd.Contracts).To(HaveLen(2))
		Expect(pd.Contracts[0].Name).To(Equal("s1"))
		Expect(pd.Contracts[0].Type).To(Equal(desc.REST))
		Expect(pd.Contracts[0].Encoding).To(Equal(kit.JSON.Tag()))
		Expect(pd.Contracts[0].Path).To(Equal("/path1"))
		Expect(pd.Contracts[0].Method).To(Equal("GET"))
		Expect(pd.Contracts[0].GroupName).To(Equal("c1"))

		Expect(pd.Contracts[1].Name).To(Equal("s2"))
		Expect(pd.Contracts[1].Type).To(Equal(desc.REST))
		Expect(pd.Contracts[1].Encoding).To(Equal(kit.JSON.Tag()))
		Expect(pd.Contracts[1].Path).To(Equal("/path2"))
		Expect(pd.Contracts[1].Method).To(Equal("POST"))
		Expect(pd.Contracts[1].GroupName).To(Equal("c1"))

		Expect(pd.Contracts[0].Request.Message.Name).To(Equal("NestedMessage"))
		Expect(pd.Contracts[0].Request.Message.Params).To(HaveLen(4))
		Expect(pd.Contracts[0].Request.Message.Params[0].Name).To(Equal("a"))
		Expect(pd.Contracts[0].Request.Message.Params[0].SubKind).To(Equal(desc.None))
		Expect(pd.Contracts[0].Request.Message.Params[0].Kind).To(Equal(desc.String))

		Expect(pd.Contracts[0].Request.Message.Params[1].Name).To(Equal("b"))
		Expect(pd.Contracts[0].Request.Message.Params[1].Kind).To(Equal(desc.Object))
		Expect(pd.Contracts[0].Request.Message.Params[1].Message.Params).To(HaveLen(4))
		Expect(pd.Contracts[0].Request.Message.Params[1].Message.Params[0].Name).To(Equal("a"))
		Expect(pd.Contracts[0].Request.Message.Params[1].Message.Params[0].Kind).To(Equal(desc.String))

	})
})
